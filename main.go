package main

import (
	"errors"
	"fmt"
	_ "image/png"
	"log"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/paran01d/psuedorace/renderer"
	"github.com/paran01d/psuedorace/spritesheet"
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
}

type segment struct {
	index  int
	p1     gamepoint
	p2     gamepoint
	color  renderer.SegmentColor
	looped bool
}

type gamepoint struct {
	world  zpoint
	camera zpoint
	screen screenpoint
}

type zpoint struct {
	x float64
	y float64
	z float64
}

type screenpoint struct {
	x     float64
	y     float64
	w     float64
	scale float64
}

type Game struct {
	config            gameConfig
	world             worldValues
	segments          []segment
	render            *renderer.Renderer
	backgroundImage   *ebiten.Image
	backgroundSprites map[string]*spritesheet.Sprite
	playerImage       *ebiten.Image
	playerSprites     map[string]*spritesheet.Sprite
	colors            map[string]renderer.SegmentColor
	skycolor          string
	treecolor         string
	fogcolor          string
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

	g.render = renderer.NewRenderer(1024, 768)

	// Load sprites
	err, backgroundImage, backgroundSprites := g.loadSpriteSheet("images/background.yml")
	if err != nil {
		log.Fatal(err)
	}
	g.backgroundImage = backgroundImage
	g.backgroundSprites = backgroundSprites

	err, playerImage, playerSprites := g.loadSpriteSheet("images/player.yml")
	if err != nil {
		log.Fatal(err)
	}
	g.playerImage = playerImage
	g.playerSprites = playerSprites

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

func (g *Game) buildTrack() {
	g.segments = make([]segment, 0)
	for n := 0; n < 500; n++ {
		color := g.colors["LIGHT"]
		if (n/g.config.rumbleLength)%2 == 0 {
			color = g.colors["DARK"]
		}

		g.segments = append(g.segments, segment{
			index: n,
			p1: gamepoint{
				world: zpoint{
					z: float64(n * g.config.segmentLength),
				},
			},
			p2: gamepoint{
				world: zpoint{
					z: float64((n + 1) * g.config.segmentLength),
				},
			},
			color: color,
		})
	}

	g.segments[g.findSegment(int(g.world.playerZ)).index+2].color = g.colors["START"]
	g.segments[g.findSegment(int(g.world.playerZ)).index+3].color = g.colors["START"]
	for n := 0; n < g.config.rumbleLength; n++ {
		g.segments[len(g.segments)-1-n].color = g.colors["FINISH"]
	}

	g.world.trackLength = len(g.segments) * g.config.segmentLength
}

func (g *Game) findSegment(z int) segment {
	if z < 0 {
		z = 0
	}
	return g.segments[z/g.config.segmentLength%len(g.segments)]
}

func (g *Game) Update() error {
	dt := (1 / ebiten.CurrentTPS())

	g.world.position = g.increase(g.world.position, g.world.speed, float64(g.world.trackLength))

	dx := dt * 2 * (g.world.speed / g.world.maxSpeed)

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

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.world.speed = g.accelerate(g.world.speed, g.world.accel, dt)
	} else if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.world.speed = g.accelerate(g.world.speed, g.world.breaking, dt)
	} else {
		g.world.speed = g.accelerate(g.world.speed, g.world.decel, dt)
	}

	if (g.world.playerX < -1 || g.world.playerX > 1) && g.world.speed > g.world.offRoadLimit {
		g.world.speed = g.accelerate(g.world.speed, g.world.offRoadDecel, dt)
	}

	g.world.playerX = g.limit(g.world.playerX, -2, 2)           // dont ever let player go too far out of bounds
	g.world.speed = g.limit(g.world.speed, 0, g.world.maxSpeed) // or exceed maxSpeed

	return nil
}

func (g *Game) limit(value, min, max float64) float64 {
	return math.Max(min, math.Min(value, max))
}

func (g *Game) accelerate(v, accel float64, dt float64) float64 {
	return v + accel*dt
}

func (g *Game) increase(start float64, increment float64, max float64) float64 {
	var result = start + increment
	for result >= max {
		result -= max
	}
	for result < 0 {
		result += max
	}
	return result
}

func (g *Game) project(gp *gamepoint, cameraX, cameraY, cameraZ, cameraDepth, width, height, roadWidth float64) {
	gp.camera.x = (gp.world.x) - cameraX
	gp.camera.y = (gp.world.y) - cameraY
	gp.camera.z = (gp.world.z) - cameraZ
	gp.screen.scale = cameraDepth / gp.camera.z
	gp.screen.x = math.Round((width / 2) + (gp.screen.scale * gp.camera.x * width / 2))
	gp.screen.y = math.Round((height / 2) - (gp.screen.scale * gp.camera.y * height / 2))
	gp.screen.w = math.Round((gp.screen.scale * roadWidth * width / 2))
}

func (g *Game) Draw(screen *ebiten.Image) {
	debugImage := ebiten.NewImage(screenWidth, screenHeight)
	ebitenutil.DebugPrint(debugImage, "Hello, World!\n")
	ebitenutil.DebugPrintAt(debugImage, fmt.Sprintf("TPS: %f Speed: %f Position: %f", ebiten.CurrentTPS(), g.world.speed, g.world.position), 50, 50)

	// draw segements
	baseSegment := g.findSegment(int(g.world.position))
	maxy := screenHeight

	screen.DrawImage(g.backgroundImage.SubImage(g.backgroundSprites["sky"].Rect()).(*ebiten.Image), nil)
	screen.DrawImage(g.backgroundImage.SubImage(g.backgroundSprites["hills"].Rect()).(*ebiten.Image), nil)
	screen.DrawImage(g.backgroundImage.SubImage(g.backgroundSprites["trees"].Rect()).(*ebiten.Image), nil)

	for n := 0; n < g.config.drawDistance; n++ {
		segment := g.segments[(baseSegment.index+n)%len(g.segments)]
		segment.looped = segment.index < baseSegment.index

		camzmodifier := 0.0
		if segment.looped {
			camzmodifier = float64(g.world.trackLength)
		}
		g.project(&segment.p1, (g.world.playerX * g.config.roadWidth), g.config.cameraHeight, g.world.position-camzmodifier, g.world.cameraDepth, screenWidth, screenHeight, g.config.roadWidth)
		g.project(&segment.p2, (g.world.playerX * g.config.roadWidth), g.config.cameraHeight, g.world.position-camzmodifier, g.world.cameraDepth, screenWidth, screenHeight, g.config.roadWidth)

		if (segment.p1.camera.z <= g.world.cameraDepth) || // behind us
			(int(segment.p2.screen.y) >= maxy) { // clip by (already rendered) segment
			continue
		}

		g.render.Segment(screenWidth, g.config.lanes,
			segment.p1.screen.x,
			segment.p1.screen.y,
			segment.p1.screen.w,
			segment.p2.screen.x,
			segment.p2.screen.y,
			segment.p2.screen.w,
			0,
			segment.color)
	}

	screen.DrawImage(g.render.Image(), nil)
	g.render.Clear()

	scale := g.world.cameraDepth / g.world.playerZ
	spriteScale := 0.3 * (1 / 128.00)
	bounce := 1.5 * rand.Float64() * (scale) * float64(g.world.resolution) * []float64{-1, 1}[rand.Intn(2)]
	op := &ebiten.DrawImageOptions{}
	destW := ((128 * scale * screenWidth) / 2) * (spriteScale * g.config.roadWidth)
	destH := ((128 * scale * screenWidth) / 2) * (spriteScale * g.config.roadWidth)

	destX := ((screenWidth - destW) / 2)     // * -0.5
	destY := (screenHeight + bounce - destH) // * -1
	ebitenutil.DebugPrintAt(debugImage, fmt.Sprintf("DestW: %f DestX: %f DestY: %f ", destW, destX, destY), 50, 100)
	op.GeoM.Scale(destW/128, destH/128)
	op.GeoM.Translate(destX, destY)
	screen.DrawImage(g.playerImage.SubImage(g.playerSprites[g.world.playerMode].Rect()).(*ebiten.Image), op)
	screen.DrawImage(debugImage, nil)

}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 1024, 768
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Hello, World!")

	rand.Seed(100)
	game := &Game{}
	game.Initialize()
	game.buildTrack()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
