package track

import (
	"log"

	"github.com/paran01d/pseudorace/renderer"
	"github.com/paran01d/pseudorace/util"
)

type Track struct {
	Length        map[string]float64
	Curve         map[string]float64
	Hill          map[string]float64
	Segments      []Segment
	RumbleLength  int
	SegmentLength int
	colors        map[string]renderer.SegmentColor
	util          *util.Util
	playerZ       float64
}

type Segment struct {
	Index       int
	P1          util.Gamepoint
	P2          util.Gamepoint
	Curve       float64
	Color       renderer.SegmentColor
	Looped      bool
	TunnelStart bool
	TunnelEnd   bool
	InTunnel    bool
}

func NewTrack(rumbleLength int, segmentLength int, playerZ float64, util *util.Util, colors map[string]renderer.SegmentColor) *Track {
	return &Track{
		Length:        map[string]float64{"none": 0, "short": 25, "medium": 50, "long": 100},
		Curve:         map[string]float64{"none": 0, "easy": 2, "medium": 4, "hard": 6},
		Hill:          map[string]float64{"none": 0, "low": 80, "medium": 140, "high": 200},
		colors:        colors,
		RumbleLength:  rumbleLength,
		SegmentLength: segmentLength,
		playerZ:       playerZ,
		util:          util,
	}
}

func (t *Track) addSegment(curve float64, y float64, tunnelStart, tunnelEnd, inTunnel bool) {
	n := len(t.Segments)

	color := t.colors["LIGHT"]
	if (n/t.RumbleLength)%2 == 0 {
		color = t.colors["DARK"]
	}

	segment := Segment{
		Index: n,
		P1: util.Gamepoint{
			World: util.Zpoint{
				Y: t.lastY(),
				Z: float64(n * t.SegmentLength),
			},
		},
		P2: util.Gamepoint{
			World: util.Zpoint{
				Y: y,
				Z: float64((n + 1) * t.SegmentLength),
			},
		},
		Color: color,
		Curve: curve,
	}

	segment.TunnelStart = tunnelStart
	segment.TunnelEnd = tunnelEnd
	segment.InTunnel = inTunnel
	log.Printf("Segment: %+v", segment)

	t.Segments = append(t.Segments, segment)

}

func (t *Track) lastY() float64 {
	if len(t.Segments) == 0 {
		return 0
	}
	return t.Segments[len(t.Segments)-1].P2.World.Y
}

func (t *Track) addRoad(enter, hold, leave, curve, y float64, tunnelStart, tunnelEnd, inTunnel bool) {
	startY := t.lastY()
	endY := startY + (y * float64(t.SegmentLength))
	total := enter + hold + leave
	setStartTunnel := false
	for n := 0.0; n < enter; n++ {
		myTunnelStart := false
		if tunnelStart && !setStartTunnel {
			setStartTunnel = true
			myTunnelStart = true
		}
		t.addSegment(t.util.EaseIn(0.0, curve, n/enter), t.util.EaseInOut(startY, endY, n/total), myTunnelStart, false, inTunnel)
	}
	for n := 0.0; n < hold; n++ {
		t.addSegment(curve, t.util.EaseInOut(startY, endY, (enter+n)/total), false, false, inTunnel)
	}
	for n := 0.0; n < leave; n++ {
		t.addSegment(t.util.EaseInOut(curve, 0.0, n/leave), t.util.EaseInOut(startY, endY, (enter+hold+n)/total), false, false, inTunnel)
	}
}

func (t *Track) addTunnel(num float64) {
	if num == 0 {
		num = t.Length["medium"]
	}
	t.addRoad(num, num, num, 0.0, 0.0, true, false, true)
	t.addRoad(num, num, num, 0.0, 0.0, false, false, true)
	t.addRoad(num, num, num, 0.0, 0.0, false, false, true)
	t.addRoad(num, num, num, 0.0, 0.0, false, false, true)
	t.addRoad(num, num, num, 0.0, 0.0, false, false, true)
	t.addRoad(num, num, num, 0.0, 0.0, false, true, true)
}

func (t *Track) addStraight(num, hill float64, tunnelStart, tunnelEnd, inTunnel bool) {
	if num == 0 {
		num = t.Length["medium"]
	}
	t.addRoad(num, num, num, 0.0, hill, tunnelStart, tunnelEnd, inTunnel)
}

func (t *Track) addCurve(num, curve, hill float64, startTunnel bool, endTunnel bool, inTunnel bool) {
	if num == 0 {
		num = t.Length["medium"]
	}
	if curve == 0 {
		curve = t.Curve["medium"]
	}
	t.addRoad(num, num, num, curve, hill, startTunnel, endTunnel, inTunnel)
}

func (t *Track) addSCurves() {
	t.addRoad(t.Length["medium"], t.Length["medium"], t.Length["medium"], -t.Curve["easy"], 0.0, false, false, false)
	t.addRoad(t.Length["medium"], t.Length["medium"], t.Length["medium"], t.Curve["medium"], 0.0, false, false, false)
	t.addRoad(t.Length["medium"], t.Length["medium"], t.Length["medium"], t.Curve["easy"], 0.0, false, false, false)
	t.addRoad(t.Length["medium"], t.Length["medium"], t.Length["medium"], -t.Curve["easy"], 0.0, false, false, false)
	t.addRoad(t.Length["medium"], t.Length["medium"], t.Length["medium"], -t.Curve["medium"], 0.0, false, false, false)
}

func (t *Track) BuildTrackWithTunnel() int {
	t.Segments = make([]Segment, 0)

	// The track
	t.addStraight(t.Length["short"], 0.0, false, false, false)
	t.addTunnel(t.Length["medium"])

	// Start and Finish markers
	// t.Segments[t.FindSegment(int(t.playerZ)).Index+2].Color = t.colors["START"]
	// t.Segments[t.FindSegment(int(t.playerZ)).Index+3].Color = t.colors["START"]
	// for n := 0; n < t.RumbleLength; n++ {
	// 	t.Segments[len(t.Segments)-1-n].Color = t.colors["FINISH"]
	// }

	return len(t.Segments) * t.SegmentLength
}

func (t *Track) BuildTrack() int {
	t.Segments = make([]Segment, 0)

	// The track
	t.addStraight(t.Length["short"]/4, 0.0, false, false, false)
	t.addStraight(t.Length["short"]/6, 0.0, true, true, true)
	t.addStraight(t.Length["short"]/4, 0.0, false, false, false)
	t.addStraight(t.Length["short"]/6, 0.0, true, true, true)
	t.addSCurves()
	t.addStraight(t.Length["long"], 0.0, false, false, false)
	t.addCurve(t.Length["medium"], t.Curve["medium"], 0.0, false, false, false)
	t.addCurve(t.Length["long"], t.Curve["medium"], 0.0, false, false, false)
	t.addStraight(0.0, 0.0, false, false, false)
	t.addSCurves()
	t.addCurve(t.Length["long"], -t.Curve["medium"], 0.0, false, false, false)
	t.addCurve(t.Length["long"], t.Curve["medium"], 0.0, false, false, false)
	t.addStraight(0.0, 0.0, true, false, true)
	t.addStraight(0.0, 0.0, false, false, true)
	t.addStraight(0.0, 0.0, false, true, true)
	t.addStraight(t.Length["short"], t.Hill["high"]*2, false, false, false)
	t.addStraight(t.Length["short"], t.Hill["high"], true, false, false)
	t.addStraight(t.Length["short"], t.Hill["high"], false, false, false)
	t.addStraight(t.Length["short"], t.Hill["high"], false, false, false)
	t.addStraight(t.Length["short"], t.Hill["low"], false, false, false)
	t.addDownhillToEnd(150)

	t.addSCurves()
	t.addCurve(t.Length["long"], -t.Curve["easy"], 0.0, false, false, false)
	t.addDownhillToEnd(0)

	// Start and Finish markers
	t.Segments[t.FindSegment(int(t.playerZ)).Index+2].Color = t.colors["START"]
	t.Segments[t.FindSegment(int(t.playerZ)).Index+3].Color = t.colors["START"]
	for n := 0; n < t.RumbleLength; n++ {
		t.Segments[len(t.Segments)-1-n].Color = t.colors["FINISH"]
	}

	return len(t.Segments) * t.SegmentLength
}

func (t *Track) BuildHillyTrack() int {
	t.Segments = make([]Segment, 0)

	t.addStraight(t.Length["short"], t.Hill["high"]*2, false, false, false)
	t.addStraight(t.Length["short"], t.Hill["high"], true, false, false)
	t.addStraight(t.Length["short"], t.Hill["high"], false, false, false)
	t.addStraight(t.Length["short"], t.Hill["high"], false, false, false)
	t.addStraight(t.Length["short"], t.Hill["low"], false, false, false)
	t.addDownhillToEnd(150)

	// Start and Finish markers
	t.Segments[t.FindSegment(int(t.playerZ)).Index+2].Color = t.colors["START"]
	t.Segments[t.FindSegment(int(t.playerZ)).Index+3].Color = t.colors["START"]
	for n := 0; n < t.RumbleLength; n++ {
		t.Segments[len(t.Segments)-1-n].Color = t.colors["FINISH"]
	}

	return len(t.Segments) * t.SegmentLength
}

func (t *Track) addDownhillToEnd(num float64) {
	if num == 0 {
		num = 200
	}
	t.addRoad(num, num, num, -t.Curve["easy"], -t.lastY()/float64(t.SegmentLength), false, false, false)
}

func (t *Track) BuildCircleTrack() int {
	t.Segments = make([]Segment, 0)
	t.addCurve(t.Length["long"], -t.Curve["medium"], 0.0, false, false, false)
	t.addCurve(t.Length["long"], -t.Curve["medium"], -t.Hill["medium"], false, false, false)
	t.addCurve(t.Length["long"], -t.Curve["medium"], 0.0, true, false, true)
	t.addCurve(t.Length["long"], -t.Curve["medium"], 0.0, false, true, true)
	t.addDownhillToEnd(0)

	// Start and Finish markers
	t.Segments[t.FindSegment(int(t.playerZ)).Index+2].Color = t.colors["START"]
	t.Segments[t.FindSegment(int(t.playerZ)).Index+3].Color = t.colors["START"]
	for n := 0; n < t.RumbleLength; n++ {
		t.Segments[len(t.Segments)-1-n].Color = t.colors["FINISH"]
	}

	return len(t.Segments) * t.SegmentLength
}
func (t *Track) FindSegment(z int) Segment {
	if z < 0 {
		z = 0
	}
	return t.Segments[z/t.SegmentLength%len(t.Segments)]
}
