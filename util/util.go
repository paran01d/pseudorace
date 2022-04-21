package util

import "math"

type Util struct {
}

type Gamepoint struct {
	World  Zpoint
	Camera Zpoint
	Screen Screenpoint
}

type Zpoint struct {
	X float64
	Y float64
	Z float64
}

type Screenpoint struct {
	X     float64
	Y     float64
	W     float64
	Scale float64
}

func NewUtil() *Util {
	return &Util{}
}

func (u *Util) Limit(value, min, max float64) float64 {
	return math.Max(min, math.Min(value, max))
}

func (u *Util) Accelerate(v, accel float64, dt float64) float64 {
	return v + accel*dt
}

func (u *Util) Increase(start float64, increment float64, max float64) float64 {
	var result = start + increment
	for result >= max {
		result -= max
	}
	for result < 0 {
		result += max
	}
	return result
}

func (u *Util) EaseIn(a, b, percent float64) float64 {
	return a + (b-a)*math.Pow(percent, 2)
}

func (u *Util) EaseOut(a, b, percent float64) float64 {
	return a + (b-a)*(1-math.Pow(1-percent, 2))
}

func (u *Util) EaseInOut(a, b, percent float64) float64 {
	return a + (b-a)*((-math.Cos(percent*math.Pi)/2)+0.5)
}

func (u *Util) Project(gp *Gamepoint, cameraX, cameraY, cameraZ, cameraDepth, width, height, roadWidth float64) {
	gp.Camera.X = (gp.World.X) - cameraX
	gp.Camera.Y = (gp.World.Y) - cameraY
	gp.Camera.Z = (gp.World.Z) - cameraZ
	gp.Screen.Scale = cameraDepth / gp.Camera.Z
	gp.Screen.X = math.Round((width / 2) + (gp.Screen.Scale * gp.Camera.X * width / 2))
	gp.Screen.Y = math.Round((height / 2) - (gp.Screen.Scale * gp.Camera.Y * height / 2))
	gp.Screen.W = math.Round((gp.Screen.Scale * roadWidth * width / 2))
}
