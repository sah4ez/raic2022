package model

import (
	"fmt"
	"io"
	"math"

	. "ai_cup_22/stream"
)

// 2 dimensional vector.
type Vec2 struct {
	// `x` coordinate of the vector
	X float64
	// `y` coordinate of the vector
	Y float64
}

func (v Vec2) IsZero() bool {
	return v.X == 0.0 && v.Y == 0.0
}

func (v Vec2) IsOne() bool {
	return v.X == 1.0 && v.Y == 1.0
}

func (v Vec2) Minus(t Vec2) Vec2 {
	return Vec2{
		X: v.X - t.X,
		Y: v.Y - t.Y,
	}
}

func (v Vec2) Noramalize() Vec2 {
	m := math.Sqrt(v.X*v.X + v.Y*v.Y)
	return Vec2{
		v.X / m,
		v.Y / m,
	}
}

func (v Vec2) Magnitude() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}

func (v Vec2) MinusU(u Unit) Vec2 {
	return v.Minus(u.Position)
}

func (v Vec2) Plus(t Vec2) Vec2 {
	return Vec2{
		X: v.X + t.X,
		Y: v.Y + t.Y,
	}
}

func (v Vec2) Mult(f float64) Vec2 {
	return Vec2{
		X: v.X * f,
		Y: v.Y * f,
	}
}

func (v Vec2) Invert() Vec2 {
	return v.Mult(-1.0)
}

func (v Vec2) Scalar(f Vec2) Vec2 {
	return Vec2{
		X: v.X * f.X,
		Y: v.Y * f.Y,
	}
}

func (v Vec2) Log() string {
	return "[" +
		fmt.Sprintf("%.f", v.X) + ":" +
		fmt.Sprintf("%.f", v.Y) +
		"]"
}

func (v1 Vec2) Distance(v2 Vec2) float64 {
	return math.Sqrt(math.Pow(float64(v2.X-v1.X), float64(2)) + math.Pow(float64(v2.Y-v1.Y), float64(2)))
}

func NewVec2(x float64, y float64) Vec2 {
	return Vec2{
		X: x,
		Y: y,
	}
}

// Read Vec2 from reader
func ReadVec2(reader io.Reader) Vec2 {
	var x float64
	x = ReadFloat64(reader)
	var y float64
	y = ReadFloat64(reader)
	return Vec2{
		X: x,
		Y: y,
	}
}

// Write Vec2 to writer
func (vec2 Vec2) Write(writer io.Writer) {
	x := vec2.X
	WriteFloat64(writer, x)
	y := vec2.Y
	WriteFloat64(writer, y)
}

// Get string representation of Vec2
func (vec2 Vec2) String() string {
	stringResult := "{ "
	stringResult += "X: "
	x := vec2.X
	stringResult += fmt.Sprint(x)
	stringResult += ", "
	stringResult += "Y: "
	y := vec2.Y
	stringResult += fmt.Sprint(y)
	stringResult += " }"
	return stringResult
}
