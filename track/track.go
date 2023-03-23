package track

import (
	"github.com/paran01d/pseudorace/renderer"
	"github.com/paran01d/pseudorace/util"
)

type Track struct {
	Length        map[string]float64
	Curve         map[string]float64
	Hills         map[string]float64
	Segments      []Segment
	RumbleLength  int
	SegmentLength int
	colors        map[string]renderer.SegmentColor
	util          *util.Util
	playerZ       float64
}

type Segment struct {
	Index  int
	P1     util.Gamepoint
	P2     util.Gamepoint
	Curve  float64
	Color  renderer.SegmentColor
	Looped bool
}

func NewTrack(rumbleLength int, segmentLength int, playerZ float64, util *util.Util) *Track {
	return &Track{
		Length: map[string]float64{"none": 0, "short": 25, "medium": 50, "long": 100},
		Curve:  map[string]float64{"none": 0, "easy": 2, "medium": 4, "hard": 6},
		Hills:  map[string]float64{"none": 0, "low": 20, "medium": 40, "high": 60, "very high": 80, "extreme": 100},
		colors: map[string]renderer.SegmentColor{
			"LIGHT":  {Road: "#6B6B6B", Grass: "#10AA10", Rumble: "#555555", Lane: "#CCCCCC"},
			"DARK":   {Road: "#696969", Grass: "#009A00", Rumble: "#BBBBBB"},
			"START":  {Road: "#fff", Grass: "#fff", Rumble: "#fff"},
			"FINISH": {Road: "#000", Grass: "#000", Rumble: "#000"},
		},
		RumbleLength:  rumbleLength,
		SegmentLength: segmentLength,
		playerZ:       playerZ,
		util:          util,
	}
}

func (t *Track) lastY() float64 {
	ret := 0.0
	if len(t.Segments) > 0 {
		ret = t.Segments[len(t.Segments)-1].P2.World.Y
	}
	return ret
}

func (t *Track) addSegment(curve float64, y float64) {
	n := len(t.Segments)

	color := t.colors["LIGHT"]
	if (n/t.RumbleLength)%2 == 0 {
		color = t.colors["DARK"]
	}

	t.Segments = append(t.Segments, Segment{
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
	})

}

func (t *Track) addRoad(enter, hold, leave, curve, y float64) {
	startY := t.lastY()
	endY := startY + (y * float64(t.SegmentLength))
	total := enter + hold + leave
	for n := 0.0; n < enter; n++ {
		t.addSegment(t.util.EaseIn(0.0, curve, n/enter), t.util.EaseInOut(startY, endY, n/total))
	}
	for n := 0.0; n < hold; n++ {
		t.addSegment(curve, t.util.EaseInOut(startY, endY, (enter+n)/total))
	}
	for n := 0.0; n < leave; n++ {
		t.addSegment(t.util.EaseInOut(curve, 0.0, n/leave), t.util.EaseInOut(startY, endY, (enter+hold+n)/total))
	}
}

func (t *Track) addStraight(num float64) {
	if num == 0 {
		num = t.Length["medium"]
	}
	t.addRoad(num, num, num, 0.0, 0.0)
}

func (t *Track) addHill(num, height float64) {
	if num == 0 {
		num = t.Length["medium"]
	}
	if height == 0 {
		height = t.Hills["medium"]
	}
	t.addRoad(num, num, num, 0.0, height)
}

func (t *Track) addCurve(num, curve, height float64) {
	if num == 0 {
		num = t.Length["medium"]
	}
	if curve == 0 {
		curve = t.Curve["medium"]
	}
	t.addRoad(num, num, num, curve, height)
}

func (t *Track) addLowRollingHills(num, height float64) {
	if num == 0 {
		num = t.Length["short"]
	}
	if height == 0 {
		height = t.Hills["low"]
	}
	t.addRoad(num, num, num, 0, height/2)
	t.addRoad(num, num, num, 0, -height)
	t.addRoad(num, num, num, 0, height)
	t.addRoad(num, num, num, 0, 0)
	t.addRoad(num, num, num, 0, height/2)
	t.addRoad(num, num, num, 0, 0)
}

func (t *Track) addSCurves() {
	t.addRoad(t.Length["medium"], t.Length["medium"], t.Length["medium"], -t.Curve["easy"], t.Hills["none"])
	t.addRoad(t.Length["medium"], t.Length["medium"], t.Length["medium"], t.Curve["medium"], t.Hills["medium"])
	t.addRoad(t.Length["medium"], t.Length["medium"], t.Length["medium"], t.Curve["easy"], t.Hills["low"])
	t.addRoad(t.Length["medium"], t.Length["medium"], t.Length["medium"], -t.Curve["easy"], t.Hills["medium"])
	t.addRoad(t.Length["medium"], t.Length["medium"], t.Length["medium"], -t.Curve["medium"], t.Hills["medium"])
}

func (t *Track) addDownhillToEnd(num float64) {
	if num == 0 {
		num = 200
	}
	t.addRoad(num, num, num, t.Curve["easy"], -t.lastY()/float64(t.SegmentLength))
}

func (t *Track) BuildTrack() int {
	t.Segments = make([]Segment, 0)

	// The track
	t.addStraight(t.Length["short"] / 4)
	t.addSCurves()
	t.addStraight(t.Length["long"])
	t.addCurve(t.Length["medium"], t.Curve["medium"], t.Hills["low"])
	t.addCurve(t.Length["long"], t.Curve["medium"], t.Hills["low"])
	t.addStraight(0.0)
	t.addSCurves()
	t.addCurve(t.Length["long"], -t.Curve["medium"], t.Hills["low"])
	t.addCurve(t.Length["long"], t.Curve["medium"], t.Hills["low"])
	t.addStraight(0.0)
	t.addSCurves()
	t.addCurve(t.Length["long"], -t.Curve["easy"], t.Hills["low"])

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

	// The track
	t.addStraight(t.Length["short"] / 2)
	t.addHill(t.Length["short"], t.Hills["low"])
	t.addLowRollingHills(0, 0)
	t.addCurve(t.Length["medium"], t.Curve["medium"], t.Hills["low"])
	t.addLowRollingHills(0, 0)
	t.addCurve(t.Length["long"], t.Curve["medium"], t.Hills["medium"])
	t.addStraight(0)
	t.addCurve(t.Length["long"], -t.Curve["medium"], t.Hills["medium"])
	t.addHill(t.Length["long"], t.Hills["high"])
	t.addCurve(t.Length["long"], -t.Curve["medium"], -t.Hills["low"])
	t.addHill(t.Length["long"], -t.Hills["medium"])
	t.addStraight(0)
	t.addDownhillToEnd(0)

	// Start and Finish markers
	t.Segments[t.FindSegment(int(t.playerZ)).Index+2].Color = t.colors["START"]
	t.Segments[t.FindSegment(int(t.playerZ)).Index+3].Color = t.colors["START"]
	for n := 0; n < t.RumbleLength; n++ {
		t.Segments[len(t.Segments)-1-n].Color = t.colors["FINISH"]
	}

	return len(t.Segments) * t.SegmentLength

}

func (t *Track) BuildCircleTrack() int {
	t.Segments = make([]Segment, 0)
	t.addCurve(t.Length["long"], -t.Curve["medium"], t.Hills["low"])
	t.addCurve(t.Length["long"], -t.Curve["medium"], 0)
	t.addCurve(t.Length["long"], -t.Curve["medium"], 0)
	t.addCurve(t.Length["long"], -t.Curve["medium"], 0)

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
