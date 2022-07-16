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

func (st *MyStrategy) NearestProj(u Unit) (Projectile, bool) {
	if len(st.projectiles) > 0 {
		sort.Sort(NewByDistanceProjectiles(u, st.projectiles))
		return st.projectiles[0], true
	}
	return Projectile{}, false
}

func (st *MyStrategy) NearestProjs(u Unit, cnt int) []Projectile {
	if len(st.projectiles) > cnt {
		sort.Sort(NewByDistanceProjectiles(u, st.projectiles))
		return st.projectiles[:cnt]
	}
	return []Projectile{}
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
