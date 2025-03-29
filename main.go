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
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
	roadWidth      float64
	rumbleLength   int
	segmentLength  int
	lanes          int
	fieldOfView    float64
	cameraHeight   float64
	drawDistance   int
	fogDensity     int
	centrifugal    float64
	drawBackground bool
	drawFog        bool
	drawPlayer     bool
	drawDebug      bool
	drawRoad       bool
	drawTunnel     bool
	drawSprite     bool
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

	carImage   *ebiten.Image
	carSprites map[string]*spritesheet.Sprite

	obstacleImage   *ebiten.Image
	obstacleSprites map[string]*spritesheet.Sprite

	billboardImage   *ebiten.Image
	billboardSprites map[string]*spritesheet.Sprite
}

func (g *Game) Initialize() {

	g.skycolor = "#72D7EE"
	g.treecolor = "#005108"
	g.fogcolor = "#005108"

	g.colors = map[string]renderer.SegmentColor{
		"LIGHT":  {Road: "#6B6B6B", Grass: "#10AA10", Rumble: "#555555", Lane: "#CCCCCC", Tunnel: "#373737", TunnelOuter: "#808080"},
		"DARK":   {Road: "#696969", Grass: "#009A00", Rumble: "#BE1B08", Tunnel: "#373737", TunnelOuter: "#808080"},
		"START":  {Road: "#ffffff", Grass: "#ffffff", Rumble: "#ffffff", Tunnel: "#000000"},
		"FINISH": {Road: "#000000", Grass: "#000000", Rumble: "#000000", Tunnel: "#000000"},
	}

	// Set config
	g.config = gameConfig{
		roadWidth:      3000,
		rumbleLength:   3,
		segmentLength:  80,
		lanes:          3,
		fieldOfView:    95,
		cameraHeight:   2200,
		drawDistance:   200,
		fogDensity:     5,
		centrifugal:    0.3,
		drawBackground: true,
		drawPlayer:     true,
		drawFog:        true,
		drawRoad:       true,
		drawDebug:      true,
		drawTunnel:     true,
		drawSprite:     true, // Always show sprites
	}

	// Setup the world
	g.world = worldValues{
		resolution:  768 / 480,
		trackLength: 0,
		cameraDepth: 1 / math.Tan((g.config.fieldOfView / 2)) * (math.Pi / 180),
		playerX:     0,
		playerMode:  "straight",
		position:    0,
		speed:       0,
		maxSpeed:    float64(100),
	}
	g.world.accel = g.world.maxSpeed / 10
	g.world.breaking = -g.world.maxSpeed
	g.world.decel = -g.world.maxSpeed / 5
	g.world.offRoadDecel = -g.world.maxSpeed / 2
	g.world.offRoadLimit = g.world.maxSpeed / 4
	g.world.playerZ = g.config.cameraHeight * g.world.cameraDepth
	// Increase the sprite scale factor to make sprites more visible with perspective
	// Use a larger scale value for sprites to be more visible
	g.world.spriteScale = 1.5
	g.world.screenScale = g.world.cameraDepth / g.world.playerZ

	g.render = renderer.NewRenderer(1024, 768, g.util)

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

	g.render.SetupBgPart(g.background)

	err, g.playerImage, g.playerSprites = g.loadSpriteSheet("images/player.yml")
	if err != nil {
		log.Fatal(err)
	}

	err, g.carImage, g.carSprites = g.loadSpriteSheet("images/cars.yml")
	if err != nil {
		log.Fatal(err)
	}

	err, g.billboardImage, g.billboardSprites = g.loadSpriteSheet("images/billboards.yml")
	if err != nil {
		log.Fatal(err)
	}

	err, g.obstacleImage, g.obstacleSprites = g.loadSpriteSheet("images/obstacles.yml")
	if err != nil {
		log.Fatal(err)
	}

	g.generateFog()
	g.bgImage = ebiten.NewImage(1024, 768)

}

func (g *Game) generateFog() {
	const fogHeight = 32
	w := screenWidth
	fogRGBA := image.NewRGBA(image.Rect(0, 0, w, fogHeight))
	for j := 0; j < fogHeight; j++ {
		a := uint32(float64(fogHeight-1-j) * 0x0f / (fogHeight - 1))
		clr := color.RGBA{0x80, 0x80, 0x80, 0xff}
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

	if inpututil.KeyPressDuration(ebiten.KeyD) == 1 {
		g.config.drawDebug = !g.config.drawDebug
	}

	if inpututil.KeyPressDuration(ebiten.KeyF) == 1 {
		g.config.drawFog = !g.config.drawFog
	}

	if inpututil.KeyPressDuration(ebiten.KeyT) == 1 {
		g.config.drawTunnel = !g.config.drawTunnel
	}

	if inpututil.KeyPressDuration(ebiten.KeyS) == 1 {
		g.config.drawSprite = !g.config.drawSprite
	}

	if inpututil.KeyPressDuration(ebiten.KeyR) == 1 {
		g.config.drawRoad = !g.config.drawRoad
	}

	if inpututil.KeyPressDuration(ebiten.KeyB) == 1 {
		g.config.drawBackground = !g.config.drawBackground
	}

	if inpututil.KeyPressDuration(ebiten.KeyP) == 1 {
		g.config.drawPlayer = !g.config.drawPlayer
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
	reversing := false

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.world.speed = g.util.Accelerate(g.world.speed, g.world.accel, dt)
	} else if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.world.speed = g.util.Accelerate(g.world.speed, g.world.breaking, dt)
		if g.world.speed < 0 {
			reversing = true
		}
	} else {
		g.world.speed = g.util.Accelerate(g.world.speed, g.world.decel, dt)
	}

	if (g.world.playerX < -1 || g.world.playerX > 1) && g.world.speed > g.world.offRoadLimit {
		g.world.speed = g.util.Accelerate(g.world.speed, g.world.offRoadDecel, dt)
	}

	if playerSegment.InTunnel {
		g.world.playerX = g.util.Limit(g.world.playerX, -0.82, 0.82) // dont ever let player go past tunnel walls
	} else {
		g.world.playerX = g.util.Limit(g.world.playerX, -2, 2) // dont ever let player go too far out of bounds
	}
	if reversing {
		g.world.speed = g.util.Limit(g.world.speed, -30, g.world.maxSpeed) // or exceed maxSpeed
	} else {
		g.world.speed = g.util.Limit(g.world.speed, 0, g.world.maxSpeed) // or exceed maxSpeed
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.White)

	// draw segements
	baseSegment := g.road.FindSegment(int(g.world.position))
	basePercent := g.util.PercentRemaining(int(g.world.position), g.config.segmentLength)

	playerSegment := g.road.FindSegment(int(g.world.position + g.world.playerZ))
	playerPercent := g.util.PercentRemaining(int(g.world.position+g.world.playerZ), g.config.segmentLength)
	playerY := g.util.Interpolate(playerSegment.P1.World.Y, playerSegment.P2.World.Y, playerPercent)

	maxy := float64(screenHeight)
	x := 0.0
	dx := -(baseSegment.Curve * basePercent)
	if g.config.drawBackground {
		g.render.Background(g.background, g.bgImage, playerY)
		screen.DrawImage(g.bgImage, nil)
	}

	segments := []renderer.SegmentDetails{}
	for n := 0; n <= g.config.drawDistance; n++ {
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
			((segment.P2.Screen.Y >= segment.P1.Screen.Y) && !segment.InTunnel) || // back face cull
			((segment.P2.Screen.Y >= maxy) && !segment.InTunnel) { // clip by (already rendered) segment
			continue
		}

		segments = append(segments, renderer.SegmentDetails{
			Index:       segment.Index,
			P1:          &segment.P1.Screen,
			P2:          &segment.P2.Screen,
			Color:       segment.Color,
			TunnelStart: segment.TunnelStart,
			TunnelEnd:   segment.TunnelEnd,
			InTunnel:    segment.InTunnel,
			Sprites:     segment.Sprites,
		})

		maxy = segment.P1.Screen.Y
	}

	// Render the segments backwards
	segments[0].PlayerSegment = true
	for i := len(segments) - 1; i >= 0; i-- {
		segment := segments[i]
		g.render.Segment(screenWidth, screenHeight, g.config.lanes, segment)
		// Draw sprites for this segment with improved tunnel clipping
		for _, sprite := range segment.Sprites {
			// Find the current segment the player is on
			playerSegment := g.road.FindSegment(int(g.world.position + g.world.playerZ))
			
			// SIMPLIFIED VISIBILITY LOGIC
			
			// Check visibility based on tunnel locations
			playerInTunnel := playerSegment.InTunnel
			spriteInTunnel := segment.InTunnel
			spriteBehindPlayer := segment.Index < playerSegment.Index
			
			// 1. If player is in a tunnel
			if playerInTunnel {
				// 1a. If sprite is in the same tunnel, show only if within width
				if spriteInTunnel {
					if sprite.Offset < -0.9 || sprite.Offset > 0.9 {
						// Skip sprites outside the tunnel walls
						continue
					}
				} else if spriteBehindPlayer {
					// 1b. If sprite is outside the tunnel but behind player
					// Skip - it's hidden by the tunnel entrance
					continue
				}
				// 1c. If sprite is outside tunnel and ahead of player, show it
				// (We're looking through the tunnel exit)
			} else {
				// 2. Player is outside tunnel
				// 2a. If sprite is in a tunnel ahead of player, hide it
				if spriteInTunnel && !spriteBehindPlayer {
					continue
				}
				
				// 2b. If sprite is in a tunnel behind player, show if within tunnel width
				if spriteInTunnel && spriteBehindPlayer {
					if sprite.Offset < -0.9 || sprite.Offset > 0.9 {
						continue
					}
				}
				
				// 2c. If sprite is outside tunnel, check for blocking tunnels
				if !spriteInTunnel {
					skipSprite := false
					// Look for tunnels between player and sprite
					for j := playerSegment.Index + 1; j < segment.Index; j++ {
						if j < len(g.road.Segments) && g.road.Segments[j].TunnelStart {
							skipSprite = true
							break
						}
					}
					if skipSprite {
						continue
					}
				}
			}
			
			// Pass the segment's perspective scale factor correctly
			// This will be inverted in the renderer to make sprites properly scale with distance
			spriteScale := segment.P1.Scale
			
			// Position sprites at the segment's screen position
			spriteX := segment.P1.X
			spriteY := segment.P1.Y
			
			// Use the exact segment road width for positioning
			segmentRoadWidth := segment.P1.W
			
			// Draw the sprite with proper perspective parameters
			g.render.Sprite(float64(screenWidth), float64(screenHeight), float64(g.world.resolution),
				segmentRoadWidth, sprite.Sprite, spriteScale, spriteX, spriteY, sprite.Offset, 0.0, maxy)
		}
	}

	roadImg := g.render.Image()
	if g.config.drawFog {
		fogop := &ebiten.DrawImageOptions{}
		fogop.GeoM.Translate(0, maxy-16)
		roadImg.DrawImage(g.fogImage, fogop)
	}
	if g.config.drawRoad {
		screen.DrawImage(roadImg, nil)
	}

	// Get the current player segment to check if we're in a tunnel
	currentSegment := g.road.FindSegment(int(g.world.position + g.world.playerZ))
	playerInTunnel := currentSegment.InTunnel
	
	// Conditional drawing order based on player's tunnel status
	if playerInTunnel {
		// When inside a tunnel:
		// 1. Draw sprites first
		if g.config.drawSprite {
			screen.DrawImage(g.render.SpriteImage(), nil)
		}
		
		// 2. Draw tunnels on top to clip sprites visible through exit
		if g.config.drawTunnel {
			screen.DrawImage(g.render.TunnelImage(), nil)
		}
	} else {
		// When outside a tunnel:
		// 1. Draw tunnels first
		if g.config.drawTunnel {
			screen.DrawImage(g.render.TunnelImage(), nil)
		}
		
		// 2. Draw sprites on top
		if g.config.drawSprite {
			screen.DrawImage(g.render.SpriteImage(), nil)
		}
	}

	g.render.Clear()

	speedPercent := g.world.speed / g.world.maxSpeed

	// Calculate bounce effect based on speed
	bounceBase := (1.5 * rand.Float64() * speedPercent * float64(g.world.resolution))
	bounceModify := []float64{-1, 1}[rand.Intn(2)]
	bounce := bounceBase * bounceModify
	
	// Get player sprite dimensions
	playerSprite := g.playerImage.SubImage(g.playerSprites[g.world.playerMode].Rect()).(*ebiten.Image)
	playerWidth := float64(playerSprite.Bounds().Dx())
	playerHeight := float64(playerSprite.Bounds().Dy())
	
	// Calculate player scaling
	playerScale := 0.65 * g.world.screenScale
	
	// Calculate player position
	destX := float64(screenWidth) / 2 // Center horizontally
	destY := float64(screenHeight) * 0.85 // Position near bottom of screen
	
	// Apply a small vertical offset for the bounce effect
	destY += bounce
	
	// Draw player sprite
	if g.config.drawPlayer {
		op := &ebiten.DrawImageOptions{}
		// Center sprite
		op.GeoM.Translate(-playerWidth/2, -playerHeight)
		// Scale appropriately
		op.GeoM.Scale(playerScale, playerScale)
		// Position on screen
		op.GeoM.Translate(destX, destY)
		// Draw the player
		screen.DrawImage(playerSprite, op)
	}
	if g.config.drawDebug {
		g.render.ResetDebug()
		g.render.DebugPrintAt(fmt.Sprintf("TPS: %f Speed: %f Position: %f desty: %f bounce: %f bounceBase: %f bounceModify: %f, res: %f", ebiten.CurrentTPS(), speedPercent, g.world.position, destY, bounce, bounceBase, bounceModify, float64(g.world.resolution)), 50, 50)
		screen.DrawImage(g.render.DebugImage(), nil)
	}
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
	track := track.NewTrack(game.config.rumbleLength, game.config.segmentLength, game.world.playerZ, util, game.colors)
	game.util = util
	game.road = track
	//game.world.trackLength = game.road.BuildCircleTrack()
	game.world.trackLength = game.road.BuildTrack(game.obstacleImage, game.obstacleSprites)
	// game.world.trackLength = game.road.BuildHillyTrack()
	//game.world.trackLength = game.road.BuildTrackWithTunnel()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
