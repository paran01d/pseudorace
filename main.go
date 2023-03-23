package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/paran01d/pseudorace/renderer"
	"github.com/paran01d/pseudorace/spritesheet"
	"github.com/paran01d/pseudorace/track"
	"github.com/paran01d/pseudorace/util"
)

const (
	screenWidth  = 1024
	screenHeight = 768
)

type gameConfig struct {
	roadWidth     float64
	rumbleLength  int
	segmentLength int
	lanes         int
	fieldOfView   float64
	cameraHeight  float64
	drawDistance  int
	fogDensity    int
	centrifugal   float64
}

type worldValues struct {
	resolution   int
	trackLength  int
	cameraDepth  float64
	playerX      float64
	playerZ      float64
	playerMode   string
	position     float64
	speed        float64
	maxSpeed     float64
	accel        float64
	breaking     float64
	decel        float64
	offRoadDecel float64
	offRoadLimit float64
	spriteScale  float64
	screenScale  float64
}

type Game struct {
	util          *util.Util
	config        gameConfig
	world         worldValues
	render        *renderer.Renderer
	background    renderer.Background
	playerImage   *ebiten.Image
	playerSprites map[string]*spritesheet.Sprite
	colors        map[string]renderer.SegmentColor
	skycolor      string
	treecolor     string
	fogcolor      string
	fogImage      *ebiten.Image
	bgImage       *ebiten.Image
	road          *track.Track
}

func (g *Game) Initialize() {

	g.skycolor = "#72D7EE"
	g.treecolor = "#005108"
	g.fogcolor = "#005108"

	g.colors = map[string]renderer.SegmentColor{
		"LIGHT":  {Road: "#6B6B6B", Grass: "#10AA10", Rumble: "#555555", Lane: "#CCCCCC"},
		"DARK":   {Road: "#696969", Grass: "#009A00", Rumble: "#BBBBBB"},
		"START":  {Road: "#fff", Grass: "#fff", Rumble: "#fff"},
		"FINISH": {Road: "#000", Grass: "#000", Rumble: "#000"},
	}

	// Set config
	g.config = gameConfig{
		roadWidth:     3000,
		rumbleLength:  3,
		segmentLength: 500,
		lanes:         3,
		fieldOfView:   100,
		cameraHeight:  2000,
		drawDistance:  300,
		fogDensity:    5,
		centrifugal:   0.3,
	}

	// Setup the world
	g.world = worldValues{
		resolution:  0,
		trackLength: 0,
		cameraDepth: 1 / math.Tan(((g.config.fieldOfView/2)*math.Pi)/180),
		playerX:     0,
		playerMode:  "straight",
		position:    0,
		speed:       0,
		maxSpeed:    float64(380),
	}
	g.world.accel = g.world.maxSpeed / 5
	g.world.breaking = -g.world.maxSpeed
	g.world.decel = -g.world.maxSpeed / 5
	g.world.offRoadDecel = -g.world.maxSpeed / 2
	g.world.offRoadLimit = g.world.maxSpeed / 4
	g.world.playerZ = g.config.cameraHeight * g.world.cameraDepth
	g.world.spriteScale = 0.3 * (1 / 128.00)
	g.world.screenScale = g.world.cameraDepth / g.world.playerZ

	g.render = renderer.NewRenderer(1024, 768)

	// Load sprites
	err, backgroundImage, backgroundSprites := g.loadSpriteSheet("images/background.yml")
	if err != nil {
		log.Fatal(err)
	}
	g.background = renderer.Background{
		Image: backgroundImage,
		Parts: []*renderer.BackgroundPart{
			{Speed: 0.1, Sprite: backgroundImage.SubImage(backgroundSprites["sky"].Rect()).(*ebiten.Image), Offset: 1408},
			{Speed: 0.2, Sprite: backgroundImage.SubImage(backgroundSprites["hills"].Rect()).(*ebiten.Image), Offset: 1408},
			{Speed: 0.3, Sprite: backgroundImage.SubImage(backgroundSprites["trees"].Rect()).(*ebiten.Image), Offset: 1408},
		},
	}

	err, playerImage, playerSprites := g.loadSpriteSheet("images/player.yml")
	if err != nil {
		log.Fatal(err)
	}
	g.playerImage = playerImage
	g.playerSprites = playerSprites

	g.generateFog()
	g.bgImage = ebiten.NewImage(1024, 768)

}

func (g *Game) generateFog() {
	const fogHeight = 16
	w := screenWidth
	fogRGBA := image.NewRGBA(image.Rect(0, 0, w, fogHeight))
	for j := 0; j < fogHeight; j++ {
		a := uint32(float64(fogHeight-1-j) * 0xff / (fogHeight - 1))
		clr := color.RGBA{0x00, 0x51, 0x08, 0xff}
		r, g, b, oa := uint32(clr.R), uint32(clr.G), uint32(clr.B), uint32(clr.A)
		clr.R = uint8(r * a / oa)
		clr.G = uint8(g * a / oa)
		clr.B = uint8(b * a / oa)
		clr.A = uint8(a)
		for i := 0; i < w; i++ {
			fogRGBA.SetRGBA(i, j, clr)
		}
	}
	g.fogImage = ebiten.NewImageFromImage(fogRGBA)
}

func (g *Game) loadSpriteSheet(file string) (error, *ebiten.Image, map[string]*spritesheet.Sprite) {
	// Load sprite sheets
	sheet, err := spritesheet.OpenAndRead(file)
	if err != nil {
		return fmt.Errorf("Could not open spritesheet: %s", err), nil, nil
	}

	img, _, err := ebitenutil.NewImageFromFile(sheet.Image)
	if err != nil {
		return fmt.Errorf("Could not open image: %+v with error: %s", sheet, err), nil, nil
	}

	return nil, img, sheet.Sprites()
}

func (g *Game) Update() error {
	var playerSegment = g.road.FindSegment(int(g.world.position + g.world.playerZ))
	tps := ebiten.CurrentTPS()
	if tps == 0 {
		tps = 60
	}
	dt := (1 / tps)
	speedPercent := (g.world.speed / g.world.maxSpeed)
	dx := dt * 2 * speedPercent
	if math.IsNaN(dx) {
		dx = 0
	}

	g.world.position = g.util.Increase(g.world.position, g.world.speed, float64(g.world.trackLength))

	for _, part := range g.background.Parts {
		part.Offset = g.util.Increase(
			part.Offset,
			part.Speed*playerSegment.Curve*speedPercent,
			2688,
		)
		if part.Offset >= 0 && part.Offset < 1 {
			part.Offset = 1408
		}
		if part.Offset <= 128 {
			part.Offset = 1408
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return errors.New("Quit pressed")
	}
	g.world.playerMode = "straight"

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.world.playerX = g.world.playerX - dx
		g.world.playerMode = "left"
	}

	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.world.playerX = g.world.playerX + dx
		g.world.playerMode = "right"
	}

	g.world.playerX = g.world.playerX - dx*speedPercent*playerSegment.Curve*g.config.centrifugal

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.world.speed = g.util.Accelerate(g.world.speed, g.world.accel, dt)
	} else if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.world.speed = g.util.Accelerate(g.world.speed, g.world.breaking, dt)
	} else {
		g.world.speed = g.util.Accelerate(g.world.speed, g.world.decel, dt)
	}

	if (g.world.playerX < -1 || g.world.playerX > 1) && g.world.speed > g.world.offRoadLimit {
		g.world.speed = g.util.Accelerate(g.world.speed, g.world.offRoadDecel, dt)
	}

	g.world.playerX = g.util.Limit(g.world.playerX, -2, 2)           // dont ever let player go too far out of bounds
	g.world.speed = g.util.Limit(g.world.speed, 0, g.world.maxSpeed) // or exceed maxSpeed

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.render.ResetDebug()

	// draw segements
	baseSegment := g.road.FindSegment(int(g.world.position))
	basePercent := g.util.PercentRemaining(int(g.world.position), g.config.segmentLength)
	playerSegment := g.road.FindSegment(int(g.world.position + g.world.playerZ))
	playerPercent := g.util.PercentRemaining(int(g.world.position+g.world.playerZ), g.config.segmentLength)
	playerY := g.util.Interpolate(playerSegment.P1.World.Y, playerSegment.P2.World.Y, playerPercent)
	maxy := screenHeight
	g.render.DebugPrintAt(fmt.Sprintf("TPS: %f Speed: %f Position: %f PlayerX: %f PlayerY: %f", ebiten.CurrentTPS(), g.world.speed, g.world.position, g.world.playerX, playerY), 50, 50)
	x := 0.0
	dx := -(baseSegment.Curve * basePercent)
	g.render.Background(g.background, g.bgImage, (0.2 * playerY))
	screen.DrawImage(g.bgImage, nil)

	for n := 0; n < g.config.drawDistance; n++ {
		segment := g.road.Segments[(baseSegment.Index+n)%len(g.road.Segments)]
		segment.Looped = segment.Index < baseSegment.Index

		camzmodifier := 0.0
		if segment.Looped {
			camzmodifier = float64(g.world.trackLength)
		}
		g.util.Project(
			&segment.P1,
			(g.world.playerX*g.config.roadWidth)-x,
			playerY+g.config.cameraHeight,
			g.world.position-camzmodifier,
			g.world.cameraDepth,
			screenWidth,
			screenHeight,
			g.config.roadWidth,
		)
		g.util.Project(
			&segment.P2,
			(g.world.playerX*g.config.roadWidth)-x-dx,
			playerY+g.config.cameraHeight,
			g.world.position-camzmodifier,
			g.world.cameraDepth,
			screenWidth,
			screenHeight,
			g.config.roadWidth,
		)

		x = x + dx
		dx = dx + segment.Curve

		if (segment.P1.Camera.Z <= g.world.cameraDepth) || // behind us
			(segment.P2.Screen.Y >= segment.P1.Screen.Y) || // back face cull
			(int(segment.P2.Screen.Y) >= maxy) { // clip by (already rendered) segment
			continue
		}

		g.render.Segment(screenWidth, g.config.lanes,
			segment.P1.Screen.X,
			segment.P1.Screen.Y,
			segment.P1.Screen.W,
			segment.P2.Screen.X,
			segment.P2.Screen.Y,
			segment.P2.Screen.W,
			segment.Color)
	}
	roadImg := g.render.Image()
	fogop := &ebiten.DrawImageOptions{}
	fogop.GeoM.Translate(0, screenHeight/2)
	//roadImg.DrawImage(g.fogImage, fogop)
	screen.DrawImage(roadImg, nil)
	g.render.Clear()

	bounce := 1.5 * rand.Float64() * (g.world.screenScale) * []float64{-1, 1}[rand.Intn(2)]
	op := &ebiten.DrawImageOptions{}
	destW := ((128 * g.world.screenScale * screenWidth) / 2) * (g.world.spriteScale * g.config.roadWidth)
	destH := ((128 * g.world.screenScale * screenWidth) / 2) * (g.world.spriteScale * g.config.roadWidth)

	destX := ((screenWidth - destW) / 2)
	destY := ((screenHeight / 2) - (g.world.cameraDepth/g.world.playerZ)*g.util.Interpolate(playerSegment.P1.Camera.Y, playerSegment.P2.Camera.Y, playerPercent)*screenHeight/2) + bounce
	op.GeoM.Scale(destW/128, destH/128)
	op.GeoM.Translate(destX, destY)
	screen.DrawImage(g.playerImage.SubImage(g.playerSprites[g.world.playerMode].Rect()).(*ebiten.Image), op)
	screen.DrawImage(g.render.DebugImage(), nil)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 1024, 768
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("pseudorace")

	rand.Seed(100)
	game := &Game{}
	game.Initialize()
	util := util.NewUtil()
	track := track.NewTrack(game.config.rumbleLength, game.config.segmentLength, game.world.playerZ, util)
	game.util = util
	game.road = track
	game.world.trackLength = game.road.BuildCircleTrack()
	// game.world.trackLength = game.road.BuildHillyTrack()
	//game.world.trackLength = game.road.BuildTrack()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
