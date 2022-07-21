package main

import (
	. "ai_cup_22/model"
	"sort"
)

func (st *MyStrategy) NearestLootWeapon(u Unit) (Loot, bool) {
	if len(st.lootsW) > 0 {
		sort.Sort(NewByDistanceLoot(u, st.lootsW))
		st.lootWpt = &st.lootsW[0]
		for _, l := range st.lootsW {
			if w, ok := l.Item.(ItemWeapon); ok && w.TypeIndex > u.WeaponIndex() {
				return l, true
			}
		}
		return Loot{}, false
	}
	return Loot{}, false
}
func (st *MyStrategy) NearestLootWeaponArc(u Unit) (Loot, bool) {
	if len(st.lootsW) > 0 {
		sort.Sort(NewByDistanceLoot(u, st.lootsW))
		st.lootWpt = &st.lootsW[0]
		for _, l := range st.lootsW {
			if w, ok := l.Item.(ItemWeapon); ok && w.TypeIndex == 2 {
				return l, true
			}
		}
		return Loot{}, false
	}
	return Loot{}, false
}

func (st *MyStrategy) NearestLootAmmo(u Unit) (Loot, bool) {
	if len(st.lootsA) > 0 {
		sort.Sort(NewByDistanceLoot(u, st.lootsA))
		return st.lootsA[0], true
	}
	return Loot{}, false
}

func (st *MyStrategy) NearestLootAmmoByTypeIndex(u Unit) (Loot, bool) {
	if len(st.lootsA) > 0 {
		sort.Sort(NewByDistanceLoot(u, st.lootsA))
		for _, a := range st.lootsA {
			if w, ok := a.Item.(ItemAmmo); ok && w.WeaponTypeIndex == u.WeaponIndex() {
				return a, true
			}
		}
		return Loot{}, false
	}
	return Loot{}, false
}

func (st *MyStrategy) NearestLootSheild(u Unit) (Loot, bool) {
	if len(st.lootsS) > 0 {
		sort.Sort(NewByDistanceLoot(u, st.lootsS))
		return st.lootsS[0], true
	}
	return Loot{}, false
}

func (st *MyStrategy) NearestAim(u Unit) (Unit, bool) {
	if len(st.aims) > 0 {
		sort.Sort(NewByDistance(u, st.aims))
		st.aim = &st.aims[0]
		return st.aims[0], true
	}
	return Unit{}, false
}

func (st *MyStrategy) NearestUnit(u Unit) (Unit, bool) {
	units := make([]Unit, len(st.units))
	if len(units) > 0 {
		sort.Sort(NewByDistance(u, units))
		for _, uu := range units {
			if uu.Id == u.Id {
				continue
			}
			return uu, true
		}
		return Unit{}, false
	}
	return Unit{}, false
}

func (st *MyStrategy) NearestProj(u Unit) (Projectile, bool) {
	if len(st.projectiles) > 0 {
		sort.Sort(NewByDistanceProjectiles(u, st.projectiles))
		return st.projectiles[0], true
	}
	return Projectile{}, false
}

func commonSegemtPrj(prj Projectile, o Unit) bool {

	p1 := Vec2{prj.Position.X - prj.Velocity.X*300, prj.Position.Y - prj.Velocity.Y*300}
	p2 := Vec2{prj.Position.X + prj.Velocity.X, prj.Position.Y + prj.Velocity.Y}

	x1 := p1.X - o.Position.X
	y1 := p1.Y - o.Position.Y
	x2 := p2.X - o.Position.X
	y2 := p2.Y - o.Position.Y
	dx := x2 - x1
	dy := y2 - y1
	R := UnitRadius() * 2.0

	//составляем коэффициенты квадратного уравнения на пересечение прямой и окружности.
	//если на отрезке [0..1] есть отрицательные значения, значит отрезок пересекает окружность
	a := dx*dx + dy*dy
	b := 2. * (x1*dx + y1*dy)
	c := x1*x1 + y1*y1 - R*R

	//а теперь проверяем, есть ли на отрезке [0..1] решения
	if -b < 0 {
		return (c < 0)
	}
	if -b < (2 * a) {
		return ((4*a*c - b*b) < 0)
	}

	return (a+b+c < 0)
}

func (st *MyStrategy) NearestProjs(u Unit) ([]Projectile, bool) {
	result := []Projectile{}
	if len(st.projectiles) > 0 {
		sort.Sort(NewByDistanceProjectiles(u, st.projectiles))

		for _, p := range st.projectiles {
			if commonSegemtPrj(p, u) {
				result = append(result, p)
			}
		}
	}
	return result, len(result) > 0
}

func NearestObstacle(u Unit) (Obstacle, bool) {
	if len(consts.Obstacles) > 0 {
		sort.Sort(NewByDistanceObstacle(u, consts.Obstacles))
		return consts.Obstacles[0], true
	}
	return Obstacle{}, false
}

func (st *MyStrategy) NearestSound(u Unit) (Sound, bool) {
	if len(st.sounds) > 0 {
		sort.Sort(NewByDistanceSound(u, st.sounds))
		return st.sounds[0], true
	}
	return Sound{}, false
}
