package renderer

import (
	"fmt"
	"math"

	"github.com/fogleman/gg"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Renderer struct {
	ctx        *gg.Context
	img        *ebiten.Image
	debugImage *ebiten.Image
}

type SegmentColor struct {
	Road   string
	Grass  string
	Rumble string
	Lane   string
}

type BackgroundPart struct {
	X16    int
	Y16    int
	Offset float64
	Speed  float64
	Sprite *ebiten.Image
}

type Background struct {
	Image *ebiten.Image
	Parts []*BackgroundPart
}

func NewRenderer(width, height int) *Renderer {
	return &Renderer{
		ctx:        gg.NewContext(width, height),
		debugImage: ebiten.NewImage(width, height),
	}
}

func (r *Renderer) Clear() {
	r.img.Clear()
	r.ctx.ClearPath()
}

func (r *Renderer) DebugPrintAt(msg string, xpos, ypos int) {
	ebitenutil.DebugPrintAt(r.debugImage, msg, xpos, ypos)
}

func (r *Renderer) DebugImage() *ebiten.Image {
	return r.debugImage
}

func (r *Renderer) ResetDebug() {
	r.debugImage.Clear()
}

func (r *Renderer) Background(background Background, dstImg *ebiten.Image, playerY float64) {

	w, h := background.Image.Size()
	//repeat := int(math.Ceil((float64(r.ctx.Width()) / float64(w)))) + 1
	repeat := 3
	r.DebugPrintAt(fmt.Sprintf("Offset: %f YOffset: %f", background.Parts[2].Offset, playerY), 50, 150)
	for pindex, part := range background.Parts {

		bgpart := ebiten.NewImage(w*repeat, h)

		// Draw bgImage on the screen repeatedly.
		for j := 0; j < repeat; j++ {
			for i := 0; i < repeat; i++ {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(w*i), float64((h * j)))
				bgpart.DrawImage(part.Sprite, op)
				ebitenutil.DebugPrintAt(bgpart, fmt.Sprintf("%d-%d", pindex, i), w*i+50, h*j+(50*pindex))
			}
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-part.Offset, playerY)
		dstImg.DrawImage(bgpart, op)
	}
}

func (r *Renderer) Segment(width, lanes int, x1, y1, w1, x2, y2, w2 float64, color SegmentColor) {
	r1 := r.rumbleWidth(w1, float64(lanes))
	r2 := r.rumbleWidth(w2, float64(lanes))
	l1 := r.laneMakerWidth(w1, float64(lanes))
	l2 := r.laneMakerWidth(w2, float64(lanes))

	// Grass
	r.ctx.SetHexColor(color.Grass)
	r.ctx.DrawRectangle(0, y2, float64(width), y1-y2)
	r.ctx.Fill()

	r.Polygon(x1-w1-r1, y1, x1-w1, y1, x2-w2, y2, x2-w2-r2, y2, color.Rumble)
	r.Polygon(x1+w1+r1, y1, x1+w1, y1, x2+w2, y2, x2+w2+r2, y2, color.Rumble)
	r.Polygon(x1-w1, y1, x1+w1, y1, x2+w2, y2, x2-w2, y2, color.Road)

	if color.Lane != "" {
		lanew1 := (w1 * 2) / float64(lanes)
		lanew2 := (w2 * 2) / float64(lanes)
		lanex1 := x1 - w1 + lanew1
		lanex2 := x2 - w2 + lanew2
		for lane := 1; lane < lanes; lanex1, lanex2, lane = lanex1+lanew1, lanex2+lanew2, lane+1 {
			r.Polygon(
				lanex1-l1/2,
				y1,
				lanex1+l1/2,
				y1,
				lanex2+l2/2,
				y2,
				lanex2-l2/2,
				y2,
				color.Lane,
			)
		}
	}

}

func (r *Renderer) Image() *ebiten.Image {
	r.img = ebiten.NewImageFromImage(r.ctx.Image())
	return r.img
}

func (r *Renderer) Polygon(x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64, x4 float64, y4 float64, color string) {
	r.ctx.MoveTo(x1, y1)
	r.ctx.LineTo(x2, y2)
	r.ctx.LineTo(x3, y3)
	r.ctx.LineTo(x4, y4)
	r.ctx.SetHexColor(color)
	r.ctx.FillPreserve()
	r.ctx.Stroke()
	r.ctx.ClearPath()
}

func (r *Renderer) rumbleWidth(projectedRoadWidth float64, lanes float64) float64 {
	return projectedRoadWidth / math.Max(6, 2*lanes)
}

func (r *Renderer) laneMakerWidth(projectedRoadWidth float64, lanes float64) float64 {
	return projectedRoadWidth / math.Max(32, 8*lanes)
}
