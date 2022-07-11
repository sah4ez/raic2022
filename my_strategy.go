package main

import (
	. "ai_cup_22/model"
	"fmt"
	"sync"
)

type MyStrategy struct {
	debugInterface *DebugInterface
	units          map[int32]Unit
	prevUnits      map[int32]Unit
	consts         Constants
	game           Game
	me             Player

	l              *sync.RWMutex
	unitOrders     map[int32]UnitOrder
	prevUnitOrders map[int32]UnitOrder
	prevLoot       map[int32]Loot
	loot           map[int32]Loot

	projectiles []Projectile
	lootsW      []Loot
	lootsA      []Loot
	lootsS      []Loot
	aims        []Unit
	prevAims    []Unit

	lootWpt  *Loot
	pickuped bool

	sounds []Sound

	usedWeapon map[int32]struct{}

	obstacles []Obstacle

	haims     map[int32]Unit
	prevHaims map[int32]Unit

	aim *Unit

	once *sync.Once
}

func NewMyStrategy(constants Constants) *MyStrategy {
	return &MyStrategy{
		units:          make(map[int32]Unit),
		prevUnits:      make(map[int32]Unit),
		consts:         constants,
		l:              new(sync.RWMutex),
		unitOrders:     make(map[int32]UnitOrder),
		prevUnitOrders: make(map[int32]UnitOrder),
		prevLoot:       make(map[int32]Loot),
		loot:           make(map[int32]Loot),
		usedWeapon:     make(map[int32]struct{}),

		aims:     make([]Unit, 0),
		prevAims: make([]Unit, 0),

		haims:     make(map[int32]Unit),
		prevHaims: make(map[int32]Unit),
		sounds:    make([]Sound, 0),

		once: new(sync.Once),

		projectiles: make([]Projectile, 0),
	}
}

func (st MyStrategy) Reset() {
	st.unitOrders = make(map[int32]UnitOrder, 0)
	st.prevLoot = make(map[int32]Loot)
	for k, v := range st.loot {
		st.prevLoot[k] = v
	}
	st.loot = make(map[int32]Loot)

	st.prevUnits = make(map[int32]Unit)
	for k, v := range st.units {
		st.prevUnits[k] = v
	}
	st.units = make(map[int32]Unit)

	st.prevAims = make([]Unit, len(st.aims))
	copy(st.prevAims, st.aims)
	st.aims = make([]Unit, 0)

	st.prevHaims = make(map[int32]Unit)
	for k, v := range st.haims {
		st.prevHaims[k] = v
	}
	st.haims = make(map[int32]Unit)
}

func (st MyStrategy) GetOrders() map[int32]UnitOrder {
	unitOrders := make(map[int32]UnitOrder)
	st.l.RLock()
	defer st.l.RUnlock()

	for k, v := range st.unitOrders {
		unitOrders[k] = v
	}

	return unitOrders
}

func (st MyStrategy) getOrder(game Game, debugInterface *DebugInterface) Order {
	st.Reset()

	st.debugInterface = debugInterface
	if debugInterface != nil {
		debugInterface.Clear()
	}
	st.game = game

	for _, p := range game.Players {
		if p.Id == game.MyId {
			st.me = p
			break
		}
	}

	st.LoadUnits(game.Units)
	st.LoadSounds()
	st.LoadProjectilse()
	st.LoadLoot()
	// st.DoTestAction(debugInterface)
	st.DoActionUnit()
	st.PrintLootInfo()

	return Order{
		UnitOrders: st.GetOrders(),
	}
}

func (st *MyStrategy) DoTestAction(di *DebugInterface) {
	for _, u := range st.units {
		// var action ActionOrder

		vecV := zeroVec
		vecD := zeroVec

		obs, ok := st.NearestObstacle(u)
		if d := u.Position.Distance(obs.Position); ok && d > 2*(st.URadius()+obs.Radius) {
			fmt.Println(">>>>", d)
			di.AddSegment(u.Position, obs.Position, prSize, black)
			vecV = obs.Position.Minus(u.Position)
			vecD = obs.Position.Minus(u.Position)
		} else {
			vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
			vecD = st.game.Zone.CurrentCenter.Minus(u.Position)
			if u.OnPoint(st.game.Zone.CurrentCenter, st.URadius()) {
				vecV = zeroVec
				vecD = rotate(vecD, 5.0)
			}
		}

		st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
	}

}

func (st *MyStrategy) LoadProjectilse() {
	st.projectiles = make([]Projectile, 0)
	for _, u := range st.game.Projectiles {
		if _, ok := st.units[u.ShooterId]; ok {
			continue
		}
		st.projectiles = append(st.projectiles, u)
	}
	return
}

func (st *MyStrategy) LoadUnits(units []Unit) {
	st.units = make(map[int32]Unit)
	st.aims = make([]Unit, 0)
	for _, u := range units {
		if u.PlayerId == st.me.Id {
			st.units[u.Id] = u
		} else {
			st.aims = append(st.aims, u)
		}
	}
	return
}

func (st *MyStrategy) LoadSounds() {
	st.sounds = make([]Sound, 0)
	for _, s := range st.game.Sounds {
		if _, ok := st.units[s.UnitId]; !ok {
			st.sounds = append(st.sounds, s)
		}
	}
}

func (st *MyStrategy) LoadLoot() {
	st.loot = make(map[int32]Loot, 0)

	st.lootsW = make([]Loot, 0)
	st.lootsA = make([]Loot, 0)
	st.lootsS = make([]Loot, 0)

	for _, u := range st.game.Loot {
		st.loot[u.Id] = u
		switch u.Item.(type) {
		case ItemWeapon:
			st.lootsW = append(st.lootsW, u)
		case ItemAmmo:
			st.lootsA = append(st.lootsA, u)
		case ItemShieldPotions:
			st.lootsS = append(st.lootsS, u)
		}
	}
}

func (st *MyStrategy) URadius() float64 {
	return st.consts.UnitRadius
}

func (st *MyStrategy) MaxUVel() float64 {
	return st.consts.MaxUnitForwardSpeed
}

func (st *MyStrategy) DoActionUnit() {

	for i, u := range st.units {
		var action ActionOrder

		vecV := zeroVec
		vecD := zeroVec

		if !st.pickuped && false && (st.game.CurrentTick < 100 || st.lootWpt != nil) {
			loot, ok := st.NearestLootWeapon(u)
			if st.lootWpt != nil {
				loot = *st.lootWpt
			}

			_, okUsed := st.usedWeapon[loot.Id]
			if ok && !okUsed {
				fmt.Println(">>>", loot.Position)
				vecV = loot.Position.Minus(u.Position).Mult(st.consts.MaxUnitForwardSpeed)
				vecD = loot.Position.Minus(u.Position)
				if ok && u.OnPoint(loot.Position, st.URadius()) {
					act := NewActionOrderPickup(loot.Id)
					action = &act
					st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
					st.usedWeapon[loot.Id] = struct{}{}
					st.lootWpt = nil
					st.pickuped = true
					fmt.Println("pickup", loot.Position, loot)
					u.ActionResult = "pickup"
					st.units[i] = u
					return
				}
				u.ActionResult = "pickupMove"
				st.units[i] = u
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
				return
			}
		}

		delta := zeroVec
		deltaRotate := 0.0
		var prj Projectile
		prjs := st.NearestProjs(u, 5)
		prjOk := len(prjs) > 0
		if prjOk {
			prj = prjs[0]
			for _, prj := range prjs {
				p2 := Vec2{prj.Position.X + prj.Velocity.X, prj.Position.Y + prj.Velocity.Y}
				d := distantion(u.Position, prj.Position)
				if d < st.URadius()*3.0 {
					pv := pointOnCircle(d, prj.Position, p2)
					deltaRotate = angle(u.Direction, prj.Position.Minus(u.Position))
					dv := distantion(pv, u.Position)
					if dist := dv - st.URadius(); dist < st.URadius() {
						vec := u.Position.Minus(pv)
						delta = Vec2{u.Position.X + vec.X*dv, u.Position.Y + vec.X*dv}
					}
				}
			}
		}

		obs, obsOk := st.NearestObstacle(u)
		if obsOk {
			d := distantion(u.Position, obs.Position)
			if d < (obs.Radius+st.URadius())*1.2 {
				pt := pointOnCircle(obs.Radius, obs.Position, obs.Position.Minus(u.Position))
				delta = delta.Plus(pt.Minus(obs.Position))
			}
		}

		prop := st.consts.Weapons[u.WeaponIndex()]
		ammoD := prop.ProjectileSpeed / prop.ProjectileLifeTime
		aim, ok := st.NearestAim(u)
		if aim.WeaponIndex() == 2 && aim.Ammo[aim.WeaponIndex()] >= 1 {
			fmt.Println("archer....")
			vecD = aim.Position.Minus(u.Position).Plus(halfVec)
			vecV = aim.Position.Minus(u.Position.Minus(aim.Position)).Mult(st.consts.MaxUnitForwardSpeed)
			act := NewActionOrderAim(true)
			action = &act
			st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
			return
		}
		if u.Ammo[u.WeaponIndex()] == 0 {
			fmt.Println("no ammo... ahhh...!!!!")
		} else if d := u.Position.Distance(aim.Position); ok && d < ammoD {
			// 90% от дистанции выстрела
			if aim.WeaponIndex() == 2 && aim.Ammo[aim.WeaponIndex()] > 1 {
				deltaDist = 1.0
			}
			vecD = aim.Position.Minus(u.Position).Plus(halfVec)
			vecV = aim.Position.Minus(u.Position)
			if d <= ammoD*deltaDist && !prjOk {
				// vecV = vecV.Mult(-1.0)
				// пытаемся сместиться в случае отступления к центру карты
				vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
			} else if d <= ammoD*deltaDist {
				vecV = u.Position.Minus(aim.Position)
			}
			vecV = vecV.Plus(delta)
			vecD = rotate(vecD, deltaRotate)
			act := NewActionOrderAim(true)
			action = &act
			st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
			u.ActionResult = "moveAttack"
			st.units[i] = u
			return
		} else if ok {
			vecV = aim.Position.Minus(u.Position).Plus(zeroVec)
			vecD = aim.Position.Minus(u.Position)
			if prjOk {
				vecV = prj.Position.Minus(u.Position)
				vecD = rotate(vecD, deltaRotate)
			}

			loot, ok := st.NearestLootSheild(u)
			if ok && u.OnPoint(loot.Position, st.URadius()) {
				act := NewActionOrderPickup(loot.Id)
				action = &act
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
				u.ActionResult = "moveAttackPickupLoot"
				st.units[i] = u
				return
			}
			st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
			u.ActionResult = "moveAttackToAim"
			st.units[i] = u
			return
		}

		if u.Ammo[u.WeaponIndex()] == 0 {
			loot, ok := st.NearestLootAmmo(u)
			la, aok := loot.Item.(ItemAmmo)
			if aok && la.WeaponTypeIndex != u.WeaponIndex() {
				fmt.Println("skip this ammo")
			} else if ok && u.OnPoint(loot.Position, st.URadius()) {
				act := NewActionOrderPickup(loot.Id)
				action = &act
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
				u.ActionResult = "pickupAmmo"
				st.units[i] = u
				return
			} else if ok {
				vecV = loot.Position.Minus(u.Position)
				vecD = loot.Position.Minus(u.Position)
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
				u.ActionResult = "pickupAmmoMove"
				st.units[i] = u
				return
			}
		}

		if u.Shield < float64(st.consts.MaxShield/0.5) {
			if u.ShieldPotions > 0 {
				act := NewActionOrderUseShieldPotion()
				action = &act
				vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
				vecD = st.game.Zone.CurrentCenter.Minus(u.Position)
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
				u.ActionResult = "useSheild"
				st.units[i] = u
				return
			} else {
				loot, ok := st.NearestLootSheild(u)
				if ok && !prjOk {
					if ok && u.OnPoint(loot.Position, st.URadius()) {
						act := NewActionOrderPickup(loot.Id)
						action = &act
						st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
						u.ActionResult = "pickupShield"
						st.units[i] = u
						return
					} else if ok {
						vecV = loot.Position.Minus(u.Position)
						vecD = loot.Position.Minus(u.Position)
						st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
						u.ActionResult = "pickupShieldMove"
						st.units[i] = u
						return
					}
				}
			}
		}

		loot, ok := st.NearestLootSheild(u)
		if ok && u.OnPoint(loot.Position, st.URadius()) {
			act := NewActionOrderPickup(loot.Id)
			action = &act
			st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
			u.ActionResult = "walkingPickup"
			st.units[i] = u
			return
		}

		vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
		vecD = st.game.Zone.CurrentCenter.Minus(u.Position)

		vecD = rotate(u.Direction, 5.0)
		st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
		u.ActionResult = "simpleWalking"
		st.units[i] = u
	}
}

func (st *MyStrategy) NewUnitOrder(u Unit, v Vec2, d Vec2, a *ActionOrder) UnitOrder {

	uo := NewUnitOrder(v, d, a)
	// if dd := st.debugInterface; dd != nil {
	// dd.AddSegment(
	// u.Position,
	// d,
	// bigLineSize,
	// red,
	// )
	// }
	return uo
}

func (st *MyStrategy) PrintLootInfo() {

	if st.debugInterface == nil {
		return
	}
	for _, u := range st.loot {
		st.debugInterface.AddPlacedText(
			u.Position,
			fmt.Sprintf("%.d, %.4f:%.4f %s", u.Id, u.Position.X, u.Position.Y, u.Item.String()),
			halfVec,
			lootSize,
			black,
		)
	}
}

func (st *MyStrategy) MoveUnit(u Unit, o UnitOrder) {
	st.PrintUnitInfo(u)
	if _, ok := st.prevUnitOrders[u.Id]; !ok {
		st.prevUnitOrders[u.Id] = o
	} else {
		st.prevUnitOrders[u.Id] = st.unitOrders[u.Id]
	}
	st.unitOrders[u.Id] = o
}

func (st *MyStrategy) debugUpdate(debugInterface DebugInterface) {
	debugInterface.Clear()
	defer debugInterface.Flush()
}

func (st *MyStrategy) PrintAimsInfo(u Unit, a Unit) {
	if st.debugInterface == nil {
		return
	}

	st.debugInterface.AddCircle(a.Position, 10, red05)
	st.debugInterface.AddSegment(
		u.Position,
		a.Position,
		lineAttackSize,
		green05,
	)
}

func (st MyStrategy) finish() {

	fmt.Println("finish")
	if st.debugInterface == nil {
		return
	}
	st.debugInterface.AddPlacedText(
		zeroVec,
		fmt.Sprintf("killed"),
		zeroVec,
		mainSize,
		black,
	)
}
