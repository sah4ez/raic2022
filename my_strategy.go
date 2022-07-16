package main

import (
	. "ai_cup_22/model"
	"fmt"
	"math"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	nextSheildTick int32

	di      *DebugInterface
	consts  Constants
	rotated bool
)

func MaxFU() float64 {
	return consts.MaxUnitForwardSpeed
}

func MaxBU() float64 {
	return consts.MaxUnitBackwardSpeed
}

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
	respAims    []Unit
	prevAims    []Unit

	unitRotate map[int32]int32

	tick int32

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
	consts = constants

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
		unitRotate:     make(map[int32]int32),

		aims:     make([]Unit, 0),
		respAims: make([]Unit, 0),
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

	di = debugInterface
	st.debugInterface = debugInterface
	if debugInterface != nil {
		debugInterface.Clear()
		di.Clear()
		di.SetAutoFlush(true)
	}
	st.game = game
	st.tick = game.CurrentTick

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

		obs, ok := NearestObstacle(u)
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
			if u.RemainingSpawnTime != nil {
				st.respAims = append(st.respAims, u)
			} else {
				st.aims = append(st.aims, u)
			}
		}
	}
	return
}

func (st *MyStrategy) LoadSounds() {
	st.sounds = make([]Sound, 0)
	for _, s := range st.game.Sounds {
		// if _, ok := st.units[s.UnitId]; !ok {
		log.Debug().Str("p", s.Position.Log()).Msg("sound")
		st.sounds = append(st.sounds, s)
		// }
	}
}

func (st *MyStrategy) LoadLoot() {
	for _, u := range st.game.Loot {
		failDistance := distantion(u.Position, st.game.Zone.CurrentCenter)
		if failDistance < st.game.Zone.CurrentRadius*0.99 {
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
}

func UnitRadius() float64 {
	return consts.UnitRadius
}

func (st *MyStrategy) URadius() float64 {
	return st.consts.UnitRadius
}

func (st *MyStrategy) MaxUVel() float64 {
	return st.consts.MaxUnitForwardSpeed
}

func LineAttackBussy(u Unit, aim Unit) (free bool) {
	if _, ok := NearestObstacle(u); !ok {
		return ok
	}

	d := distantion(aim.Position, u.Position)
	for _, o := range consts.Obstacles {
		if d := distantion(o.Position, u.Position); d > consts.ViewDistance {
			break
		}
		if o.CanShootThrough {
			continue
		}
		dAO := distantion(aim.Position, o.Position)
		dUO := distantion(u.Position, o.Position)
		if dAO+dUO < d+o.Radius {
			fmt.Println("o>", o.Position, dAO, dUO, o.Radius, d)
			return true
		}
	}

	return false
}

func newWalking(l zerolog.Logger, vecV Vec2, vecD Vec2) zerolog.Logger {
	return l.With().
		Str("vecV", vecV.Log()).
		Str("vecD", vecD.Log()).
		Logger()
}

func checkProject(l zerolog.Logger, prjPt Vec2, u Unit, aim Unit) (Vec2, zerolog.Logger) {
	// если по нам прилет
	if u.OnPoint(prjPt, UnitRadius()*2) {
		vecV1 := rotatePoints(prjPt, u.Position, 180.0)
		vecV := vecV1.Minus(u.Position).Mult(MaxFU()) //по тапкам...
		l := l.With().
			Str("prjPt", prjPt.Log()).
			Str("vecV1", vecV1.Log()).
			Int32("aimWI", aim.WeaponIndex()).
			Logger()
		return vecV, l
	}
	return Vec2{}, l
}

func (st *MyStrategy) DoActionUnit() {
	for _, u := range st.units {
		l := log.With().
			Int32("id", u.Id).
			Int32("tick", st.game.CurrentTick).
			Str("pos", u.Position.Log()).
			Str("st_v", u.Velocity.Log()).
			Str("st_d", u.Direction.Log()).
			Logger()

		var action ActionOrder

		vecV := zeroVec
		vecD := zeroVec

		failDistance := distantion(u.Position, st.game.Zone.CurrentCenter)
		if failDistance > st.game.Zone.CurrentRadius*0.99 {
			vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
			prj, prjOk := st.NearestProj(u)
			prjPt := prjectilePointPjr(u, prj)

			if prjOk {
				if u.OnPoint(prjPt, st.URadius()*2) {
					vecV1 := rotatePoints(prjPt, u.Position, 180.0)
					l = log.With().
						Str("prj", prj.Position.Log()).
						Str("vecV1", vecV1.Log()).
						Str("prjPt", prjPt.Log()).
						Logger()
					vecV = vecV1.Minus(vecV).Mult(MaxFU()) //по тапкам...
					failDistance := distantion(vecV, st.game.Zone.CurrentCenter)
					if failDistance > st.game.Zone.CurrentRadius*0.99 {
						vecV1 := rotatePoints(vecV, prjPt, 180.0)
						vecV = vecV1.Minus(vecV).Mult(MaxFU()) //по тапкам...
					}
				}
			}

			l = newWalking(l, vecV, vecD)
			l.Debug().Msg("walking to center")
			st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
			return
		}

		rt, ok := st.unitRotate[u.Id]
		if ok && rt > st.game.CurrentTick {
			act := NewActionOrderAim(true)
			action = &act
			vecD = rotate(u.Direction, math.Pi)
			st.MoveUnit(u, st.NewUnitOrder(u, zeroVec, vecD, &action))
			l = newWalking(l, vecV, vecD)
			l.Debug().Msg("rotate")
			return
		}
		sound, soundOk := st.NearestSound(u)
		soundProp := st.consts.Sounds[sound.TypeIndex]

		if !st.pickuped && false && (st.game.CurrentTick < 50 || st.lootWpt != nil) {
			loot, ok := st.NearestLootWeapon(u)
			if st.lootWpt != nil {
				loot = *st.lootWpt
			}

			_, okUsed := st.usedWeapon[loot.Id]
			if ok && !okUsed {
				l = l.With().
					Stringer("loot", loot).
					Str("loot", loot.Position.Log()).
					Logger()
				vecV = loot.Position.Minus(u.Position).Mult(st.consts.MaxUnitForwardSpeed)
				vecD = loot.Position.Minus(u.Position)
				if ok && u.OnPoint(loot.Position, st.URadius()) {
					act := NewActionOrderPickup(loot.Id)
					action = &act
					l = newWalking(l, vecV, vecD)
					l.Debug().Msg("pickup")
					st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
					st.usedWeapon[loot.Id] = struct{}{}
					st.lootWpt = nil
					st.pickuped = true
					return
				}
				l = newWalking(l, vecV, vecD)
				l.Debug().Msg("pickupMove")
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
				return
			}
		}

		delta := zeroVec
		deltaRotate := 0.0
		var prjPoint *Vec2
		var prj Projectile
		prj, prjOk := st.NearestProj(u)
		if prjOk {
			p2 := Vec2{prj.Position.X + prj.Velocity.X, prj.Position.Y + prj.Velocity.Y}
			d := distantion(u.Position, prj.Position)
			pv := pointOnCircle(d, prj.Position, p2)
			prjPoint = &pv
			if d <= st.URadius()*2.0 {
				pv := pointOnCircle(d, prj.Position, p2)
				deltaRotate += angle(u.Direction, prj.Position.Minus(u.Position))
				dv := distantion(pv, u.Position)
				if dist := dv - st.URadius(); dist <= st.URadius() {
					vec := pv.Minus(u.Position)
					vec = vec.Scalar(Vec2{st.consts.MaxUnitBackwardSpeed, st.consts.MaxUnitBackwardSpeed})
					delta = Vec2{u.Position.X + vec.X*dv, u.Position.Y + vec.X*dv}
					l = newWalking(l, vecV, vecD)
					l.Debug().
						Str("delta", delta.Log()).
						Str("pv", pv.Log()).
						Msg("under attck")
				}
			}
			// }
		}

		loot, sheildOk := st.NearestLootSheild(u)
		if u.Shield < st.consts.MaxShield && nextSheildTick < st.game.CurrentTick {

			if u.ShieldPotions == st.consts.MaxShieldPotionsInInventory {
				l.Debug().Msg("full shield")
			} else if u.ShieldPotions > 0 && nextSheildTick < st.game.CurrentTick {
				act := NewActionOrderUseShieldPotion()
				action = &act
				vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
				vecD = st.game.Zone.CurrentCenter.Minus(u.Position)
				if sheildOk {
					vecV = loot.Position.Minus(u.Position)
					vecV = vecV.Mult(st.consts.MaxUnitForwardSpeed)
					vecD = loot.Position.Plus(u.Position)
				}
				nextSheildTick = st.tick + int32(st.consts.TicksPerSecond/st.consts.ShieldPotionUseTime)
				vecV = vecV.Plus(delta)
				vecD = rotate(u.Position, math.Pi)
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))

				l = newWalking(l, vecV, vecD)
				l.Debug().Msg("useSheild")
				return
			} else {
				if sheildOk && u.OnPoint(loot.Position, st.URadius()) {
					act := NewActionOrderPickup(loot.Id)
					action = &act
					vecV = loot.Position.Minus(u.Position)
					vecV = vecV.Minus(delta)
					st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
					l = newWalking(l, vecV, vecD)
					l.Debug().Msg("pickupShield")
					return
				} else if sheildOk {
					vecV = loot.Position.Minus(u.Position)
					vecD = loot.Position.Minus(u.Position)
					vecV = vecV.Mult(st.consts.MaxUnitForwardSpeed)
					vecV = vecV.Plus(delta)
					st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
					l = newWalking(l, vecV, vecD)
					l.Debug().Msg("pickupShieldMove")
					return
				}
			}
		}

		aim, aimOk := st.NearestAim(u)
		prop := st.consts.Weapons[u.WeaponIndex()]
		ammoD := prop.ProjectileSpeed / prop.ProjectileLifeTime
		if u.Ammo[u.WeaponIndex()] == 0 {
			l.Debug().Msg("no ammo... ahhh...!!!!")
		} else if d := u.Position.Distance(aim.Position); aimOk && d < ammoD { // можем стрелять и есть цель
			// направляем вектор в позицию относительно скорости
			// vecD = aim.Position.Minus(u.Position).Plus(aim.Direction.Scalar(aim.Velocity))
			vecD = aim.Position.Minus(u.Position) //.Plus(aim.Direction.Scalar(aim.Velocity))

			vecV = aim.Position.Minus(u.Position)
			// идем по умолчанию от цели
			// 90% от дистанции выстрела и не под прицелом
			if d <= ammoD*deltaDist && !prjOk {
				// vecV = vecV.Mult(-1.0)
				// пытаемся сместиться в случае отступления к центру карты
				if sheildOk {
					vecsV := loot.Position.Minus(u.Position)
					vecV = vecsV.Mult(MaxBU())
				} else {
					vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
				}
				// 90% от дистанции выстрела и под прицелом
			} else if d <= ammoD*deltaDist && prjOk {
				// лучник и второй тип
				vecV = u.Position.Minus(aim.Position).Mult(MaxBU()) // пытаемся максимально отойти от этих ...
			}
			if prjPoint != nil {
				vecV, l = checkProject(l, *prjPoint, u, aim)
			}
			if soundOk && u.OnPoint(sound.Position, st.URadius()) {
				l.Debug().Str("soudnName", soundProp.Name).Msg("under fire sound")
				if soundProp.Name == "BowHit" {
					vecD = sound.Position
				}
			}
			busy := LineAttackBussy(u, aim)
			act := NewActionOrderAim(!busy)
			action = &act
			if busy && ok {
				st.unitRotate[u.Id] = st.game.CurrentTick + 10
				vecD = rotate(vecD, math.Pi)
			}
			if aim.WeaponIndex() == 2 {
				vecV2 := st.game.Zone.CurrentCenter.Minus(vecV)
				vecV2 = vecV2.Noramalize()
				vecV = vecV.Minus(vecV2.Mult(3.0))
			}
			if aim.WeaponIndex() == 1 {
				// todo надо отключать прицеливание и давать по тапкам.... с криками ай-яй-яй ))
				vecV2 := st.game.Zone.CurrentCenter.Minus(vecV)
				vecV2 = vecV2.Noramalize()
				vecV = vecV.Minus(vecV2.Mult(3.0))
			}
			st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
			l = newWalking(l, vecV, vecD)
			l.Debug().Msg("moveAttack")
			return
		} else if aimOk {
			vecV = aim.Position.Minus(u.Position)
			vecD = aim.Position.Minus(u.Position)

			loot, ok := st.NearestLootSheild(u)
			if ok && u.OnPoint(loot.Position, st.URadius()) {
				act := NewActionOrderPickup(loot.Id)
				action = &act
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
				l = newWalking(l, vecV, vecD)
				l.Debug().Msg("moveAttackPickupLoot")
				return
			}

			if prjPoint != nil {
				vecV, l = checkProject(l, *prjPoint, u, aim)
			}

			if d := u.Position.Distance(aim.Position); d < ammoD { // можем стрелять и есть цель
				// 90% от дистанции выстрела и не под прицелом
				if d <= ammoD*deltaDist && !prjOk {
					// vecV = vecV.Mult(-1.0)
					// пытаемся сместиться в случае отступления к центру карты
					if sheildOk {
						vecsV := loot.Position.Minus(u.Position)
						vecV = vecsV.Mult(MaxBU())
					} else {
						vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
					}
				}
			}
			// 90% от дистанции выстрела и под прицелом
			st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
			l = newWalking(l, vecV, vecD)
			l.Debug().Msg("moveAttackToAim")
			return
		}

		if u.Ammo[u.WeaponIndex()] <= int32(float64(prop.MaxInventoryAmmo)*float64(0.05)) {
			loot, ok := st.NearestLootAmmo(u)
			la, aok := loot.Item.(ItemAmmo)
			if aok && la.WeaponTypeIndex != u.WeaponIndex() {
				l.Debug().
					Str("loot", loot.Position.Log()).
					Msg("skip this ammo")
			} else if ok && u.OnPoint(loot.Position, 2.0*st.URadius()) {
				vecV = loot.Position.Minus(u.Position)
				vecD = loot.Position.Minus(u.Position)
				vecV = vecV.Mult(MaxFU())
				vecV = vecV.Plus(delta)
				act := NewActionOrderPickup(loot.Id)
				action = &act
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
				l = newWalking(l, vecV, vecD)
				l.Debug().Msg("pickupAmmo")
				return
			} else if ok {
				vecV = loot.Position.Minus(u.Position)
				vecD = loot.Position.Minus(u.Position)
				vecV = vecV.Mult(MaxFU())
				vecV = vecV.Plus(delta)
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
				l = newWalking(l, vecV, vecD)
				l.Debug().Msg("pickupAmmoMove")
				return
			}
		}

		{
			loot, ok := st.NearestLootSheild(u)
			if ok && u.OnPoint(loot.Position, st.URadius()) {
				act := NewActionOrderPickup(loot.Id)
				action = &act
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
				l = newWalking(l, vecV, vecD)
				l.Debug().Msg("walkingPickup")
				return
			}
		}
		if u.OnPoint(st.game.Zone.CurrentCenter, st.game.Zone.CurrentRadius/2.0) {
			{
				loot, ok := st.NearestLootSheild(u)
				if ok && u.OnPoint(loot.Position, st.URadius()) {
					act := NewActionOrderPickup(loot.Id)
					action = &act
					st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
					l = newWalking(l, vecV, vecD)
					l.Debug().Msg("walkingPickup sheild in center")
					return
				}
			}
			{
				loot, ok := st.NearestLootAmmo(u)
				if ok && u.OnPoint(loot.Position, st.URadius()) && u.Ammo[u.WeaponIndex()] == prop.MaxInventoryAmmo {
					act := NewActionOrderPickup(loot.Id)
					action = &act
					st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
					l = newWalking(l, vecV, vecD)
					l.Debug().Msg("walkingPickup ammo in center")
					return
				}
			}
			{
				weaponLoot, ok := st.NearestLootWeapon(u)
				if ok && u.OnPoint(loot.Position, st.URadius()) {
					weapon := weaponLoot.Item.(ItemWeapon)
					if u.WeaponIndex() > weapon.TypeIndex && u.Ammo[weapon.TypeIndex] == prop.MaxInventoryAmmo {
						act := NewActionOrderPickup(loot.Id)
						action = &act
						st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
						l = newWalking(l, vecV, vecD)
						l.Debug().Msg("walkingPickup weapon in center")
						return
					}
				}
			}
		}

		vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
		vecD = st.game.Zone.CurrentCenter.Minus(u.Position)

		// todo собирать лут
		vecD = rotate(u.Direction, 5.0)
		st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
		l = newWalking(l, vecV, vecD)
		l.Debug().Msg("simpleWalking")
	}
}

func (st *MyStrategy) NewUnitOrder(u Unit, v Vec2, d Vec2, a *ActionOrder) UnitOrder {

	uo := NewUnitOrder(v, d, a)
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
