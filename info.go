package main

import (
	. "ai_cup_22/model"
	"fmt"

	"github.com/rs/zerolog/log"
)

func (st *MyStrategy) PrintUnitInfo(debugInterface *DebugInterface, u Unit) {
	if debugInterface == nil {
		return
	}
	st.debugLock.Lock()
	defer st.debugLock.Unlock()

	debugInterface.AddPlacedText(
		zeroVec,
		fmt.Sprintf("%.4f, %.4f:%.4f", u.Health, u.Position.X, u.Position.Y),
		zeroVec,
		mainSize,
		black,
	)

	// ct := st.game.Zone.CurrentCenter
	// r := st.game.Zone.CurrentRadius
	// st.debugInterface.AddRect(Vec2{X: ct.X - r, Y: ct.Y - r}, Vec2{X: ct.X + r, Y: ct.Y + r}, black)
	info := []string{
		fmt.Sprintf("   ID: %d", st.game.MyId),
		fmt.Sprintf("  MSP: %d", st.consts.MaxShieldPotionsInInventory),
		fmt.Sprintf(" tick: %d", st.game.CurrentTick),
		fmt.Sprintf("sTick: %d", nextSheildTick),
		fmt.Sprintf("units: %d", len(st.aims)),
		fmt.Sprintf("  prj: %d", len(st.projectiles)),
		fmt.Sprintf("loots: a:%d  w:%d  s:%d", len(st.lootsA), len(st.lootsW), len(st.lootsS)),
	}

	for _, s := range st.sounds {
		soundD := distantion(s.Position, u.Position)
		soundProp := st.consts.Sounds[s.TypeIndex]
		if soundD <= soundProp.Distance {
			debugInterface.AddRing(s.Position, soundRingRadius, soundRingSize, red025)
		} else {
			debugInterface.AddRing(s.Position, soundRingRadius, soundRingSize, blue25)
		}
	}

	info = append(info, fmt.Sprintf("uid: %d", u.Id))
	info = append(info, fmt.Sprintf("action: %s", u.ActionResult))
	info = append(info, fmt.Sprintf("p: %.2f : %.2f", u.Position.X, u.Position.Y))
	info = append(info, fmt.Sprintf("v: %.2f : %.2f", u.Velocity.X, u.Velocity.Y))
	info = append(info, fmt.Sprintf("d: %.2f : %.2f", u.Direction.X, u.Direction.Y))
	if u.RemainingSpawnTime != nil {
		info = append(info, fmt.Sprintf("%d r: %.2f", u.Id, *u.RemainingSpawnTime))
	}

	info = append(info, fmt.Sprintf("%d weapon: %d", u.Id, u.WeaponIndex()))
	for i, a := range u.Ammo {
		info = append(info, fmt.Sprintf("%d a: %d/ %d", u.Id, i, a))
	}
	info = append(info, fmt.Sprintf("%d ns: %d", u.Id, u.NextShotTick))
	info = append(info, fmt.Sprintf("%d Aim: %.4f", u.Id, u.Aim))
	info = append(info, fmt.Sprintf("%d Sheilds: %d", u.Id, u.ShieldPotions))

	// obs, ok := st.NearestObstacle(u)
	// if ok {
	// pv := pointOnCircle(obs.Radius, obs.Position, u.Position)
	//
	// st.debugInterface.AddCircle(pv, prSize, black)
	// st.debugInterface.AddSegment(u.Position, obs.Position, prSize, black)
	// }
	// }

	if w := st.lootWpt; w != nil {
		debugInterface.AddSegment(u.Position, w.Position, prSize, black05)
	}

	a, ok := st.NearestAim(u)
	if ok {
		debugInterface.AddCircle(a.Position, 1.0*twoSize, black25)
		o, busy := LineAttackBussy(u, a)
		debugInterface.AddCircle(o.Position, 1.0*twoSize, black)
		if busy {
			debugInterface.AddCircle(o.Position, 0.5*twoSize, red)
		}
	}
	for _, u := range st.aims {
		debugInterface.AddSegment(u.Position, u.Position.Plus(u.Velocity), prSize, black05)
	}

	scale := 0.2
	greenLine := map[int32]Unit{}
	p, ok := st.NearestProj(u)
	if ok {
		greenLine[p.Id] = u
	}
	for _, p := range st.projectiles {
		color := red025
		p1 := Vec2{p.Position.X - p.Velocity.X*scale, p.Position.Y - p.Velocity.Y*scale}
		p2 := Vec2{p.Position.X + p.Velocity.X, p.Position.Y + p.Velocity.Y}
		if uu, ok := greenLine[p.Id]; ok {
			color = green
			d := distantion(uu.Position, p.Position)
			pv := pointOnCircle(d, p.Position, p2)
			debugInterface.AddRing(p.Position, d, prSize, black)
			debugInterface.AddCircle(pv, bigLineSize, color)
			vecV1 := rotatePoints(pv, u.Position, 180.0)
			debugInterface.AddCircle(vecV1, lootSize, color)
		}
		debugInterface.AddSegment(p1, p2, prSize, color)
	}

	prjs, ok := st.NearestProjs(u)

	if ok {

		info = append(info, fmt.Sprintf("%d under prj: %d", u.Id, len(prjs)))
		for _, p := range prjs {

			// v1 := u.Position.Minus(p.Position)
			// v2 := u.Position.Minus(u.Velocity)
			// vuVec := math.Abs(angle(v1, v2))
			// if vuVec < 45 {
			debugInterface.AddCircle(p.Position, 2.0*prSize, blue05)
			v := u.Position.Minus(p.Position).Noramalize().Mult(MaxBU())
			debugInterface.AddSegment(p.Position, p.Position.Plus(v), prSize, blue05)
			// st.debugInterface.AddSegment(u.Position, u.Position.Minus(v), prSize, blue)
			debugInterface.AddCircle(v, 2.0*prSize, blue05)
			// }
			d := distantion(u.Position, p.Position)
			if p.WeaponTypeIndex == 2 || p.WeaponTypeIndex == 1 {
				vecV1 := Vec2{}
				p2 := Vec2{p.Position.X + p.Velocity.X*300.0, p.Position.Y + p.Velocity.Y*300.0}
				pv := pointOnCircle(d, p.Position, p2)
				// dv := distantion(pv, u.Position)
				// if dist := dv - UnitRadius(); math.Abs(dist) <= 2.0*UnitRadius() {
				vec := u.Position.Plus(pv).Noramalize()
				vecV1.X = vecV1.X + vec.X
				vecV1.Y = vecV1.Y + vec.Y
				// }
				debugInterface.AddSegment(u.Position, u.Position.Plus(vecV1.Mult(MaxBU())), 0.5*prSize, black)
				continue
			}
		}

		vecV1, _ := checkProjects(log.Logger, prjs, u)
		debugInterface.AddSegment(u.Position, u.Position.Plus(vecV1.Mult(MaxBU())), 0.5*prSize, blue)
	}

	debugInterface.AddSegment(
		u.Position,
		u.Position.Plus(u.Velocity),
		prSize,
		black,
	)
	debugInterface.AddSegment(
		u.Position,
		u.Position.Plus(u.Direction),
		lineSize,
		black25,
	)
	for i, msg := range info {
		debugInterface.AddPlacedText(
			Vec2{u.Position.X, u.Position.Y + float64(i)*float64(mainSize+1.0)},
			msg,
			Vec2{1.0, 1.0},
			mainSize,
			black,
		)
	}
}
