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

func (st *MyStrategy) NearestProjs(u Unit) ([]Projectile, bool) {
	if len(st.projectiles) > 0 {
		result := make([]Projectile, 0)
		sort.Sort(NewByDistanceProjectiles(u, st.projectiles))

		for _, p := range st.projectiles {
			if shooter, ok := st.hAims[p.ShooterId]; ok {
				distToProject := distantion(shooter.Position, p.Position)
				prjOk := shooter.OnPoint(u.Position, distToProject-st.URadius())

				if ok := u.OnPoint(p.Position, 0.8*ViewDU()); ok && !prjOk {
					result = append(result, p)
				}
			} else {
				if ok := u.OnPoint(p.Position, 0.8*ViewDU()); ok {
					result = append(result, p)
				}
			}

		}
		if len(result) == 0 {
			return nil, false
		}
		return result, true
	}
	return nil, false
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
