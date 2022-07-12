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

type ByDistanceProjectiles struct {
	cur Unit
	a   []Projectile
}

func NewByDistanceProjectiles(cur Unit, a []Projectile) ByDistanceProjectiles {
	return ByDistanceProjectiles{
		cur: cur,
		a:   a,
	}
}

func (a ByDistanceProjectiles) Len() int      { return len(a.a) }
func (a ByDistanceProjectiles) Swap(i, j int) { a.a[i], a.a[j] = a.a[j], a.a[i] }
func (a ByDistanceProjectiles) Less(i, j int) bool {
	return distantion(a.cur.Position, a.a[i].Position) < distantion(a.cur.Position, a.a[j].Position)
}

type ByDistanceObstacle struct {
	cur Unit
	a   []Obstacle
}

func NewByDistanceObstacle(cur Unit, a []Obstacle) ByDistanceObstacle {
	return ByDistanceObstacle{
		cur: cur,
		a:   a,
	}
}

func (a ByDistanceObstacle) Len() int      { return len(a.a) }
func (a ByDistanceObstacle) Swap(i, j int) { a.a[i], a.a[j] = a.a[j], a.a[i] }
func (a ByDistanceObstacle) Less(i, j int) bool {
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

// distance from point (px, py) to line segment (x1, y1, x2, y2)
func distPointToLine(p Vec2, l1 Vec2, l2 Vec2) (float64, Vec2) {
	dx := l2.X - l1.X
	dy := l2.Y - l1.Y

	length := math.Sqrt(math.Pow(dx, 2.0) + math.Pow(dy, 2.0))

	dx = dx / length
	dy = dy / length

	point := dx*(p.X-l1.X) + dy*(p.Y-l1.Y)

	if point < 0 {
		dx, dy = p.X-l1.X, p.Y-l1.Y
		return math.Sqrt(dx*dx + dy*dy), l1
	} else if point > length {
		dx, dy = p.X-l2.X, p.Y-l2.Y
		return math.Sqrt(dx*dx + dy*dy), l2
	}
	return math.Abs(dy*(p.X-l1.X) - dx*(p.Y-l1.Y)), Vec2{l1.X + dx*point, l1.Y + dy*point}
}

func distPointToLine2(p, v Vec2) float64 {
	return math.Sqrt((v.X*v.X - p.X*p.X) + (v.Y*v.Y - p.Y*p.Y))
}

func angle(v1 Vec2, v2 Vec2) float64 {
	return math.Atan2(v2.Y, v2.X) - math.Atan2(v1.Y, v1.X)
}

func pointOnCircle(r float64, p1 Vec2, p2 Vec2) Vec2 {

	dx := p2.X - p1.X
	dy := p2.Y - p1.Y
	l := math.Sqrt(dx*dx + dy*dy)
	dx = dx / l
	dy = dy / l

	return Vec2{X: p1.X + dx*r, Y: p1.Y + dy*r}

}

func rotatePoints(point Vec2, center Vec2, angle float64) Vec2 {
	angle = (angle) * (math.Pi / angle) // Convert to radians
	var rotatedX = math.Cos(angle)*(point.X-center.X) - math.Sin(angle)*(point.Y-center.Y) + center.X
	var rotatedY = math.Sin(angle)*(point.X-center.X) + math.Cos(angle)*(point.Y-center.Y) + center.Y
	return Vec2{rotatedX, rotatedY}
}
