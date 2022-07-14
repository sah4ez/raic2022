package main

import (
	. "ai_cup_22/model"
	"fmt"
)

func (st *MyStrategy) PrintUnitInfo(u Unit) {
	if st.debugInterface == nil {
		return
	}

	st.debugInterface.Clear()
	defer st.debugInterface.Flush()

	st.debugInterface.AddPlacedText(
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
		st.debugInterface.AddRing(s.Position, soundRingRadius, soundRingSize, black05)
		fmt.Println(">>>>", s.Position)
	}
	for _, u := range st.units {

		info = append(info, fmt.Sprintf("action: %s", u.ActionResult))
		info = append(info, fmt.Sprintf("p: %.2f : %.2f", u.Position.X, u.Position.Y))
		info = append(info, fmt.Sprintf("v: %.2f : %.2f", u.Velocity.X, u.Velocity.Y))
		info = append(info, fmt.Sprintf("d: %.2f : %.2f", u.Direction.X, u.Direction.Y))

		info = append(info, fmt.Sprintf("%d weapon: %d", u.Id, u.WeaponIndex()))
		info = append(info, fmt.Sprintf("%d ammo: %d", u.Id, u.Ammo[u.WeaponIndex()]))
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
	}

	if w := st.lootWpt; w != nil {
		st.debugInterface.AddSegment(u.Position, w.Position, prSize, black05)
	}

	if u := st.aim; u != nil {
		st.debugInterface.AddCircle(u.Position, 2.0*twoSize, black25)
	}
	for _, u := range st.aims {
		st.debugInterface.AddSegment(u.Position, u.Position.Plus(u.Velocity), prSize, black05)
	}

	for i, msg := range info {
		st.debugInterface.AddPlacedText(
			Vec2{u.Position.X, u.Position.Y + float64(i)*float64(mainSize+1.0)},
			msg,
			Vec2{1.0, 1.0},
			mainSize,
			black,
		)
	}
	scale := 0.2
	greenLine := map[int32]Unit{}
	for _, u := range st.units {
		p, ok := st.NearestProj(u)
		if ok {
			greenLine[p.Id] = u
		}
	}
	for _, p := range st.projectiles {
		color := red025
		p1 := Vec2{p.Position.X - p.Velocity.X*scale, p.Position.Y - p.Velocity.Y*scale}
		p2 := Vec2{p.Position.X + p.Velocity.X, p.Position.Y + p.Velocity.Y}
		if uu, ok := greenLine[p.Id]; ok {
			color = green
			// p1 := Vec2{p.Position.X - p.Velocity.X + 300, p.Position.Y - p.Velocity.Y + 300}
			// p2 := Vec2{p.Position.X + p.Velocity.X + 300, p.Position.Y + p.Velocity.Y + 300}
			d := distantion(uu.Position, p.Position)
			pv := pointOnCircle(d, p.Position, p2)
			st.debugInterface.AddRing(p.Position, d, prSize, black)
			st.debugInterface.AddCircle(pv, bigLineSize, color)
			vecV1 := rotatePoints(pv, u.Position, 180.0)
			st.debugInterface.AddCircle(vecV1, lootSize, color)

			// vec := u.Position.Plus(p.Position.Minus(pv))
			// st.debugInterface.AddSegment(pv, Vec2{u.Position.X + vec.X*100, u.Position.Y + vec.Y*100}, prSize, blue25)
		}
		st.debugInterface.AddSegment(p1, p2, prSize, color)
	}
	for _, u := range st.units {
		st.debugInterface.AddSegment(
			u.Position,
			u.Position.Plus(u.Velocity),
			prSize,
			black,
		)
		st.debugInterface.AddSegment(
			u.Position,
			u.Position.Plus(u.Direction),
			lineSize,
			black25,
		)
		// st.debugInterface.AddSegment(
		// u.Position,
		// Vec2{u.Position.X + u.Velocity.X, u.Position.Y + u.Velocity.X},
		// mainSize,
		// green,
		// )
		// st.debugInterface.AddSegment(
		// u.Position,
		// u.Velocity,
		// lineSize,
		// blue,
		// )
		// st.debugInterface.AddSegment(
		// u.Position,
		// u.Direction,
		// lineSize,
		// green,
		// )
	}
}
