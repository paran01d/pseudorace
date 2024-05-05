package renderer

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/paran01d/pseudorace/util"
)

var ()

type Renderer struct {
	img           *ebiten.Image
	debugImage    *ebiten.Image
	util          *util.Util
	whiteImage    *ebiten.Image
	whiteSubImage *ebiten.Image
	bgpart        *ebiten.Image
}

type SegmentColor struct {
	Road   string
	Grass  string
	Rumble string
	Lane   string
	Tunnel string
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

func NewRenderer(width, height int, util *util.Util) *Renderer {
	whiteImage := ebiten.NewImage(3, 3)

	// whiteSubImage is an internal sub image of whiteImage.
	// Use whiteSubImage at DrawTriangles instead of whiteImage in order to avoid bleeding edges.
	whiteSubImage := whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

	whiteImage.Fill(color.White) //RGBA{0, 78, 8, 0})

	return &Renderer{
		debugImage:    ebiten.NewImage(width, height),
		img:           ebiten.NewImage(width, height),
		util:          util,
		whiteImage:    whiteImage,
		whiteSubImage: whiteSubImage,
	}
}

func (r *Renderer) Clear() {
	r.img.Clear()
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

func (r *Renderer) SetupBgPart(background Background) {
	w, h := background.Image.Size()
	r.bgpart = ebiten.NewImage(w*3, h)
}

func (r *Renderer) Background(background Background, dstImg *ebiten.Image, playerY float64) {

	w, h := background.Image.Size()
	repeat := 3
	for pindex, part := range background.Parts {

		// Draw bgImage on the screen repeatedly.
		for j := 0; j < repeat; j++ {
			for i := 0; i < repeat; i++ {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(w*i), float64((h*j)+pindex*80))
				r.bgpart.DrawImage(part.Sprite, op)
				ebitenutil.DebugPrintAt(r.bgpart, fmt.Sprintf("%d-%d-%f", pindex, i, part.Offset), w*i+50, h*j+(50*pindex))
			}
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-part.Offset, -50.0*part.Speed*(playerY*0.0001))
		dstImg.DrawImage(r.bgpart, op)
		r.bgpart.Clear()
	}
}

type SegmentDetails struct {
	P1          *util.Screenpoint
	P2          *util.Screenpoint
	Color       SegmentColor
	TunnelStart bool
	TunnelEnd   bool
	InTunnel    bool
}

func (r *Renderer) Segment(width, height, lanes int, sd SegmentDetails) {
	r1 := r.rumbleWidth(sd.P1.W, float64(lanes))
	r2 := r.rumbleWidth(sd.P2.W, float64(lanes))
	l1 := r.laneMakerWidth(sd.P1.W, float64(lanes))
	l2 := r.laneMakerWidth(sd.P2.W, float64(lanes))

	// Grass
	// mx1 = 0, my1 = y2
	// mx2 = width, my2 = y2
	// mx3 = width, my3 = y2+(y1-y2)
	// mx4 = 0, my4 = y2+(y1-y2)

	// First side of road
	if sd.InTunnel {
		if sd.TunnelStart {
			// Draw tunnel entrance wall
			r.Polygon(0, sd.P1.Y, sd.P1.X-sd.P1.W, sd.P1.Y, sd.P1.X-sd.P1.W, sd.P1.CielingY-20, 0, sd.P1.CielingY-20, sd.Color.Tunnel)
		}
		// Left Wall
		r.Polygon(sd.P1.X-sd.P1.W, sd.P1.Y, sd.P1.X-sd.P1.W, sd.P1.CielingY, sd.P2.X-sd.P2.W, sd.P2.CielingY, sd.P2.X-sd.P2.W, sd.P2.Y, sd.Color.Tunnel)
		if sd.TunnelStart {
			// Draw tunnel entrance cieling
			r.Polygon(sd.P1.X-sd.P1.W, sd.P1.CielingY, sd.P1.X-sd.P1.W, sd.P1.CielingY-20, sd.P2.X+sd.P1.W, sd.P1.CielingY-20, sd.P2.X+sd.P1.W, sd.P1.CielingY, sd.Color.Tunnel)
		}
		// cieling
		r.Polygon(sd.P1.X-sd.P1.W, sd.P1.CielingY, sd.P1.X+sd.P1.W, sd.P1.CielingY, sd.P2.X+sd.P2.W, sd.P2.CielingY, sd.P2.X-sd.P2.W, sd.P2.CielingY, sd.Color.Tunnel)
		// Road
		r.Polygon(sd.P1.X-sd.P1.W, sd.P1.Y, sd.P1.X+sd.P1.W, sd.P1.Y, sd.P2.X+sd.P2.W, sd.P2.Y, sd.P2.X-sd.P2.W, sd.P2.Y, sd.Color.Road)
		// Right Wall
		//r.Polygon(float64(width), sd.P2.Y, sd.P2.X+sd.P2.W+r2, sd.P2.Y, sd.P1.X+sd.P1.W+r1, sd.P1.Y-float64(height), float64(width), sd.P2.Y-(float64(height)), sd.Color.Tunnel)
	} else {
		// Grass
		r.Polygon(0, sd.P2.Y, sd.P1.X-sd.P2.W, sd.P2.Y, sd.P1.X-sd.P1.W, sd.P2.Y+(sd.P1.Y-sd.P2.Y), 0, sd.P2.Y+(sd.P1.Y-sd.P2.Y), sd.Color.Grass)
		// Road
		r.Polygon(sd.P1.X-sd.P1.W-r1, sd.P1.Y, sd.P1.X-sd.P1.W, sd.P1.Y, sd.P2.X-sd.P2.W, sd.P2.Y, sd.P2.X-sd.P2.W-r2, sd.P2.Y, sd.Color.Rumble)
		r.Polygon(sd.P1.X-sd.P1.W, sd.P1.Y, sd.P1.X+sd.P1.W, sd.P1.Y, sd.P2.X+sd.P2.W, sd.P2.Y, sd.P2.X-sd.P2.W, sd.P2.Y, sd.Color.Road)
		r.Polygon(sd.P1.X+sd.P1.W+r1, sd.P1.Y, sd.P1.X+sd.P1.W, sd.P1.Y, sd.P2.X+sd.P2.W, sd.P2.Y, sd.P2.X+sd.P2.W+r2, sd.P2.Y, sd.Color.Rumble)
		// Grass
		r.Polygon(float64(width), sd.P2.Y, sd.P2.X+sd.P2.W+r2, sd.P2.Y, sd.P1.X+sd.P1.W+r1, sd.P1.Y, float64(width), sd.P2.Y+(sd.P1.Y-sd.P2.Y), sd.Color.Grass)
	}

	if sd.Color.Lane != "" {
		lanew1 := (sd.P1.W * 2) / float64(lanes)
		lanew2 := (sd.P2.W * 2) / float64(lanes)
		lanex1 := sd.P1.X - sd.P1.W + lanew1
		lanex2 := sd.P2.X - sd.P2.W + lanew2
		for lane := 1; lane < lanes; lanex1, lanex2, lane = lanex1+lanew1, lanex2+lanew2, lane+1 {
			r.Polygon(
				lanex1-l1/2,
				sd.P1.Y,
				lanex1+l1/2,
				sd.P1.Y,
				lanex2+l2/2,
				sd.P2.Y,
				lanex2-l2/2,
				sd.P2.Y,
				sd.Color.Lane,
			)
		}
	}
}

func (r *Renderer) Image() *ebiten.Image {
	return r.img
}

func (r *Renderer) Polygon(x1, y1, x2, y2, x3, y3, x4, y4 float64, color string) {
	path := vector.Path{}
	path.MoveTo(float32(x1), float32(y1))
	path.LineTo(float32(x2), float32(y2))
	path.LineTo(float32(x3), float32(y3))
	path.LineTo(float32(x4), float32(y4))
	path.Close()

	red, green, blue, _ := r.util.ParseHexColor(color)

	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].ColorR = float32(red) / float32(0xff)
		vs[i].ColorG = float32(green) / float32(0xff)
		vs[i].ColorB = float32(blue) / float32(0xff)
		vs[i].ColorA = 1
	}

	op := &ebiten.DrawTrianglesOptions{}
	op.AntiAlias = true

	r.img.DrawTriangles(vs, is, r.whiteSubImage, op)
}

func (r *Renderer) rumbleWidth(projectedRoadWidth float64, lanes float64) float64 {
	return projectedRoadWidth / math.Max(6, 2*lanes)
}

func (r *Renderer) laneMakerWidth(projectedRoadWidth float64, lanes float64) float64 {
	return projectedRoadWidth / math.Max(32, 8*lanes)
}
