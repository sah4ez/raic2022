package main

import (
	. "ai_cup_22/model"
	"sort"
)

func (st *MyStrategy) NearestLootWeapon(u Unit) (Loot, bool) {
	if len(st.lootsW) > 0 {
		sort.Sort(NewByDistanceLoot(u, st.lootsW))
		return st.lootsW[0], true
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
		return st.aims[0], true
	}
	return Unit{}, false
}
