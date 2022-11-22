package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
	bgImage *ebiten.Image
	repeat  int
)

func init() {
	// Decode an image from the image file's byte slice.
	// Now the byte slice is generated with //go:generate for Go 1.15 or older.
	// If you use Go 1.16 or newer, it is strongly recommended to use //go:embed to embed the image file.
	// See https://pkg.go.dev/embed for more details.
	img, _, err := image.Decode(bytes.NewReader(images.Tile_png))
	if err != nil {
		log.Fatal(err)
	}
	bgImage = ebiten.NewImageFromImage(img)
	w, _ := bgImage.Size()
	repeat = int(math.Ceil((float64(screenWidth) / float64(w)))) + 1
	fmt.Printf("Repeat: %d", repeat)
}

type viewport struct {
	x16 int
	y16 int
}

func (p *viewport) Move() {
	w, _ := bgImage.Size()
	maxX16 := w
	//maxY16 := h * 16

	p.x16 += 1
	//p.y16 += h / 32
	p.x16 %= maxX16
	//p.y16 %= maxY16
}

func (p *viewport) Position() (int, int) {
	return p.x16, p.y16
}

type Game struct {
	viewport viewport
}

func (g *Game) Update() error {
	g.viewport.Move()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	x16, y16 := g.viewport.Position()
	offsetX, offsetY := float64(x16), float64(y16)

	// Draw bgImage on the screen repeatedly.
	w, h := bgImage.Size()
	for j := 0; j < repeat; j++ {
		for i := 0; i < repeat; i++ {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(w*i), float64(h*j))
			op.GeoM.Translate(offsetX, offsetY)
			screen.DrawImage(bgImage, op)
		}
	}

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("TPS: %0.2f x16: %d, xoffset: %f, bgImage w: %d", ebiten.CurrentTPS(), x16, offsetX, w), 0, 50)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("screenwidth: %d Repeat: %d", screenWidth, repeat), 0, 150)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Infinite Scroll (Ebiten Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
