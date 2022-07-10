package main

import (
	"math"
	"math/rand"

	. "ai_cup_22/model"
)

type ByDistance struct {
	cur Unit
	a   []Unit
}

func NewByDistance(cur Unit, a []Unit) ByDistance {
	return ByDistance{
		cur: cur,
		a:   a,
	}
}

func (a ByDistance) Len() int      { return len(a.a) }
func (a ByDistance) Swap(i, j int) { a.a[i], a.a[j] = a.a[j], a.a[i] }
func (a ByDistance) Less(i, j int) bool {
	return distantion(a.cur.Position, a.a[i].Position) < distantion(a.cur.Position, a.a[j].Position)
}

type ByDistanceLoot struct {
	cur Unit
	a   []Loot
}

func NewByDistanceLoot(cur Unit, a []Loot) ByDistanceLoot {
	return ByDistanceLoot{
		cur: cur,
		a:   a,
	}
}

func (a ByDistanceLoot) Len() int      { return len(a.a) }
func (a ByDistanceLoot) Swap(i, j int) { a.a[i], a.a[j] = a.a[j], a.a[i] }
func (a ByDistanceLoot) Less(i, j int) bool {
	return distantion(a.cur.Position, a.a[i].Position) < distantion(a.cur.Position, a.a[j].Position)
}

func distantion(v1 Vec2, v2 Vec2) float64 {
	return math.Sqrt(math.Pow(float64(v2.X-v1.X), float64(2)) + math.Pow(float64(v2.Y-v1.Y), float64(2)))
}

func rotate(o Vec2, a float64) Vec2 {
	return Vec2{
		X: o.X*math.Cos(a) + o.Y*math.Sin(a),
		Y: o.Y*math.Cos(a) - o.X*math.Sin(a),
	}
}

// from -1.0 to 1.0
func randDirection() float64 {
	return float64(rand.Intn(2) - 1)
}
