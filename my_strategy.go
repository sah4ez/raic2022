package main

import (
	. "ai_cup_22/model"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
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

func MaxUR() float64 {
	return consts.UnitRadius
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

	var out io.Writer = os.Stdout
	if true {
		out = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	log.Logger = zerolog.New(out).With().Timestamp().Logger()
	// if cfg.ReportCaller {
	// inlog = inlog.Caller()
	// }
	// if cfg.LogStack {
	// inlog = inlog.Stack()
	// }

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
	// st.prevLoot = make(map[int32]Loot)
	// for k, v := range st.loot {
	// st.prevLoot[k] = v
	// }
	// st.loot = make(map[int32]Loot)

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
	st.DoActionUnit2()
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
	// st.loot = make(map[int32]Loot, 0)
	//
	// st.lootsW = make([]Loot, 0)
	// st.lootsA = make([]Loot, 0)
	// st.lootsS = make([]Loot, 0)

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
			log.Debug().
				Str("o", o.Position.Log()).
				Str("u", u.Position.Log()).
				Float64("dao", dAO).
				Float64("duo", dUO).
				Float64("r", o.Radius).
				Float64("d", d).
				Msg("busy")
			return true
		}
	}

	return false
}

var (
	toCenter     = false
	toCenterTick = int32(0)
)

func (st *MyStrategy) DoActionUnit() {
	for _, u := range st.units {
		l := log.With().
			Str("pos", u.Position.Log()).
			Int32("unitID", u.Id).
			Int32("t", st.game.CurrentTick).
			Logger()

		var action ActionOrder

		vecV := zeroVec
		vecD := zeroVec

		failDistance := distantion(u.Position, st.game.Zone.CurrentCenter)
		if failDistance > st.game.Zone.CurrentRadius*0.99 || (toCenter && toCenterTick > st.game.CurrentTick) {
			vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
			prj, prjOk := st.NearestProj(u)
			prjPt := prjectilePointPjr(u, prj)
			if prjOk {
				if u.OnPoint(prjPt, st.URadius()*2) {
					vecV1 := rotatePoints(prjPt, u.Position, 180.0)
					vecV = vecV1.Minus(vecV).Mult(MaxFU()) //по тапкам...
					l.Debug().Str("vecV", vecV.Log()).Str("prjPt", prjPt.Log()).Msg("prj on unit")
					failDistance := distantion(vecV, st.game.Zone.CurrentCenter)
					if failDistance > st.game.Zone.CurrentRadius*0.99 {
						vecV1 := rotatePoints(vecV, prjPt, 180.0)
						vecV = vecV1.Minus(vecV).Mult(MaxFU()) //по тапкам...
					}
				}
			}

			if toCenterTick == st.game.CurrentTick {
				toCenter = false
			} else {
				toCenterTick = st.game.CurrentTick + 10
				toCenter = true
			}
			st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
			l.Debug().Str("vecV", vecV.Log()).Str("prjPt", prjPt.Log()).Msg("walking to center")
			return
		}

		rt, ok := st.unitRotate[u.Id]
		if ok && rt > st.game.CurrentTick {
			act := NewActionOrderAim(true)
			action = &act
			vecD = rotate(u.Direction, math.Pi)
			st.MoveUnit(u, st.NewUnitOrder(u, zeroVec, vecD, &action))
			return
		}

		aim, aimOk := st.NearestAim(u)
		if loot, ok := st.NearestLootWeapon(u); ok {
			wp := loot.Item.(ItemWeapon)
			d := distantion(loot.Position, u.Position)
			points := u.Ammo[wp.TypeIndex]
			prop := st.consts.Weapons[wp.TypeIndex]
			if d < 3*st.URadius() && wp.TypeIndex > u.WeaponIndex() && !aimOk && points < prop.MaxInventoryAmmo {
				vecV = loot.Position.Minus(u.Position).Mult(st.consts.MaxUnitForwardSpeed)
				vecD = loot.Position.Minus(u.Position)
				if ok && u.OnPoint(loot.Position, st.URadius()) {
					act := NewActionOrderPickup(loot.Id)
					action = &act
					st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
					st.usedWeapon[loot.Id] = struct{}{}

					l.Debug().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("pickupWP")
					return
				}
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
				l.Debug().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("pickupMoveWP")
				return
			} else {
				prop := st.consts.Weapons[wp.TypeIndex]
				if u.Ammo[wp.TypeIndex] < prop.MaxInventoryAmmo {
					loot, ok := st.NearestLootAmmo(u)
					la, aok := loot.Item.(ItemAmmo)
					if aok {
						l = log.With().Str("ammo", loot.Position.Log()).Logger()
					}
					if aok && la.WeaponTypeIndex != u.WeaponIndex() {
						l.Log().Msg("skip this ammo")
					} else if ok && u.OnPoint(loot.Position, st.URadius()) {
						prop := st.consts.Weapons[u.WeaponIndex()]
						if u.Ammo[u.WeaponIndex()] < prop.MaxInventoryAmmo {
							vecV = loot.Position.Minus(u.Position)
							vecD = loot.Position.Minus(u.Position)
							vecV = vecV.Mult(MaxFU())
							act := NewActionOrderPickup(loot.Id)
							action = &act
							st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
							l.Log().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("pickupAmmo")
							return
						}
					} else if ok {
						vecV = loot.Position.Minus(u.Position)
						vecD = loot.Position.Minus(u.Position)
						vecV = vecV.Mult(MaxFU())
						st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
						u.ActionResult = "pickupAmmoMove"
						l.Log().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("pickupAmmoMove")
						return
					}
				}
			}
		}

		if !st.pickuped && false && (st.game.CurrentTick < 50) {
			loot, ok := st.NearestLootWeapon(u)

			_, okUsed := st.usedWeapon[loot.Id]
			if ok && !okUsed {
				l = l.With().Str("loot", loot.Position.Log()).Logger()
				vecV = loot.Position.Minus(u.Position).Mult(st.consts.MaxUnitForwardSpeed)
				vecD = loot.Position.Minus(u.Position)
				if ok && u.OnPoint(loot.Position, st.URadius()) {
					act := NewActionOrderPickup(loot.Id)
					action = &act
					st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
					st.usedWeapon[loot.Id] = struct{}{}

					l.Debug().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("pickup")
					return
				}
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
				l.Debug().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("pickupMove")
				return
			}
		}

		delta := zeroVec
		deltaRotate := 0.0
		var prjPoint *Vec2
		var prj Projectile
		prj, prjOk := st.NearestProj(u)
		// prjOk := len(prjs) > 0
		l = l.With().Bool("prjOk", prjOk).Str("prj", prj.Position.Log()).Logger()
		if prjOk {
			// prj = prjs[0]
			// for _, prj := range prjs {
			p2 := Vec2{prj.Position.X + prj.Velocity.X, prj.Position.Y + prj.Velocity.Y}
			d := distantion(u.Position, prj.Position)
			pv := pointOnCircle(d, prj.Position, p2)
			prjPoint = &pv
			if d <= st.URadius()*2.0 {
				pv := pointOnCircle(d, prj.Position, p2)
				deltaRotate += angle(u.Direction, prj.Position.Minus(u.Position))
				dv := distantion(pv, u.Position)
				if dist := dv - st.URadius(); dist < st.URadius() {
					vec := pv.Minus(u.Position)
					vec = vec.Scalar(Vec2{st.consts.MaxUnitBackwardSpeed, st.consts.MaxUnitBackwardSpeed})
					delta = Vec2{u.Position.X + vec.X*dv, u.Position.Y + vec.X*dv}
					l.Log().Str("pv", pv.Log()).Float64("dist", d).Msg("under attack")
				}
			}
			// }
		}

		loot, sheildOk := st.NearestLootSheild(u)
		if u.Shield < st.consts.MaxShield && nextSheildTick < st.game.CurrentTick && !aimOk {

			l = l.With().Int32("sps", u.ShieldPotions).Logger()
			if u.ShieldPotions == st.consts.MaxShieldPotionsInInventory {
				l.Log().Msg("full shield")
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

				l.Log().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("useSheild")
				return
			} else {
				if ok && !prjOk && u.Health > st.consts.UnitHealth*0.9 {
					if sheildOk && u.OnPoint(loot.Position, st.URadius()) {
						act := NewActionOrderPickup(loot.Id)
						action = &act
						vecV = loot.Position.Minus(u.Position)
						// vecD = loot.Position.Minus(oneVec)
						vecV = vecV.Minus(delta)
						st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
						l.Log().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("pickupShield")
						return
					} else if sheildOk {
						vecV = loot.Position.Minus(u.Position)
						vecD = loot.Position.Minus(u.Position)
						vecV = vecV.Mult(st.consts.MaxUnitForwardSpeed)
						vecV = vecV.Plus(delta)
						st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
						l.Log().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("pickupShieldMove")
						return
					}
				}
			}
		}

		currentCentere := st.game.Zone.CurrentCenter
		cause := ""
		prop := st.consts.Weapons[u.WeaponIndex()]
		ammoD := prop.ProjectileSpeed / prop.ProjectileLifeTime
		if d := u.Position.Distance(aim.Position); aimOk && d < ammoD {
			pv := nearestPoint(u.Position, currentCentere)
			vecV := pv.Noramalize().Mult(-1 * MaxFU())
			vecV = u.Position.Plus(vecV).Invert()

			// идем по умолчанию от цели
			// 90% от дистанции выстрела и не под прицелом
			if d <= ammoD*deltaDist && !prjOk {
				// vecV = vecV.Mult(-1.0)
				// пытаемся сместиться в случае отступления к центру карты
				if sheildOk {
					vecsV := loot.Position.Minus(u.Position)
					vecV = vecsV.Mult(MaxBU())
					cause = "sheildOK"
				} else {
					vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
					cause = "not  sheildOK"
				}
				// 90% от дистанции выстрела и под прицелом
			} else if d <= ammoD*deltaDist && prjOk {
				// лучник и второй тип
				vecV = u.Position.Minus(aim.Position).Mult(MaxBU()) // пытаемся максимально отойти от этих ...
				cause = "with projecte"
			}
			if prjPoint != nil {
				prjPt := *prjPoint
				// если по нам прилет
				if u.OnPoint(prjPt, st.URadius()*2) {
					cause = "underk attack"
					vecV1 := rotatePoints(prjPt, u.Position, 180.0)
					vecV = vecV1.Minus(u.Position).Mult(MaxFU()) //по тапкам...
				}
			}
			busy := LineAttackBussy(u, aim)
			act := NewActionOrderAim(!busy)
			action = &act
			if busy && ok {
				cause = "busy rotate"
				st.unitRotate[u.Id] = st.game.CurrentTick + 10
				// vecD = rotate(vecD, math.Pi)
				// vecD = rotate(u.Direction, 5.0)
				loot, ok := st.NearestLootSheild(u)
				if ok && u.OnPoint(loot.Position, st.URadius()) {
					act := NewActionOrderPickup(loot.Id)
					action = &act
					vecV = loot.Position.Minus(u.Position)
					vecV = vecV.Minus(delta)
					st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
					u.ActionResult = "moveAttackPickupLoot"
					l.Log().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("moveAttackPickupLoot")
					return
				}

			}
			st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))

			l.Log().Str("cause", cause).Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("moveAttack")
			return
		} else if aimOk {

			if loot, ok := st.NearestLootWeapon(u); ok {
				wp := loot.Item.(ItemWeapon)
				d := distantion(loot.Position, u.Position)
				points := u.Ammo[wp.TypeIndex]
				prop := st.consts.Weapons[wp.TypeIndex]
				if d < 1.5*st.URadius() && wp.TypeIndex > u.WeaponIndex() && points < prop.MaxInventoryAmmo {
					vecV = loot.Position.Minus(u.Position).Mult(st.consts.MaxUnitForwardSpeed)
					vecD = loot.Position.Minus(u.Position)
					if ok && u.OnPoint(loot.Position, st.URadius()) {
						act := NewActionOrderPickup(loot.Id)
						action = &act
						st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
						st.usedWeapon[loot.Id] = struct{}{}

						l.Debug().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("pickupWP")
						return
					}
					st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
					l.Debug().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("pickupMoveWP")
					return
				}
			}

			cause := "ammo"
			vecV = aim.Position.Minus(u.Position)
			vecD = aim.Position.Minus(u.Position)

			loot, ok := st.NearestLootSheild(u)
			if ok && u.OnPoint(loot.Position, st.URadius()) {
				act := NewActionOrderPickup(loot.Id)
				action = &act
				vecV = loot.Position.Minus(u.Position)
				vecV = vecV.Minus(delta)
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
				u.ActionResult = "moveAttackPickupLoot"
				l.Log().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("moveAttackPickupLoot")
				return
			}

			if prjPoint != nil {
				prjPt := *prjPoint
				// если по нам прилет
				if u.OnPoint(prjPt, st.URadius()*2) {
					cause = "under prject"
					vecV1 := rotatePoints(prjPt, u.Position, 180.0)
					vecV = vecV1.Minus(u.Position).Mult(MaxFU()) //по тапкам...
				}
			}

			if d := u.Position.Distance(aim.Position); d < ammoD { // можем стрелять и есть цель
				// 90% от дистанции выстрела и не под прицелом
				if d <= ammoD*deltaDist && !prjOk {
					if sheildOk {
						vecsV := loot.Position.Minus(u.Position)
						vecV = vecsV.Mult(MaxBU())
						cause = "sheildOK"
					} else {
						vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
						cause = "not  sheildOK"
					}
				}
			}
			// 90% от дистанции выстрела и под прицелом
			st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
			l.Log().Str("cause", cause).Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("moveAttackToAim")
			return
		}

		{
			loot, ok := st.NearestLootSheild(u)
			if ok && u.OnPoint(loot.Position, st.URadius()) {
				act := NewActionOrderPickup(loot.Id)
				vecV = loot.Position.Minus(u.Position)
				vecV = vecV.Minus(delta)
				action = &act
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
				l.Log().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("walkingPickup")
				return
			}
		}

		vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
		vecD = st.game.Zone.CurrentCenter.Minus(u.Position)

		vecD = rotate(u.Direction, 5.0)
		st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
		u.ActionResult = "simpleWalking"
		l.Log().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("simpleWalking")
	}
}

var (
	u1       = 1.0
	u1Switch = int32(0)
)

func nearestPoint(p1 Vec2, p2 Vec2) Vec2 {
	pp2 := p1.Plus(p2)
	d := distantion(p1, p2)
	pv := pointOnCircle(d, p2, pp2)
	return pv
}

func OnPoint(p1 Vec2, p Vec2, radius float64) bool {
	return distantion(p1, p) < radius
}

func VelocityBusy(u Unit) (free bool) {
	if _, ok := NearestObstacle(u); !ok {
		return ok
	}

	for _, o := range consts.Obstacles {

		d := distantion(o.Position, u.Position)
		if d > consts.ViewDistance {
			break
		}
		if d < o.Radius+MaxUR() {
			log.Debug().Str("o", o.Position.Log()).Float64("mxa", MaxUR()).Float64("d", d).Float64("r", o.Radius).Msg("obs")
			return true
		}
	}

	return false
}

func (st *MyStrategy) DoActionUnit2() {

	for _, u := range st.units {
		l := log.With().
			Str("pos", u.Position.Log()).
			Int32("unitID", u.Id).
			Int32("t", st.game.CurrentTick).
			Logger()
		var action ActionOrder
		currentCentere := st.game.Zone.CurrentCenter

		pv := nearestPoint(u.Position, currentCentere)
		vecV := pv.Noramalize().Mult(-1 * MaxFU())
		vecV = u.Position.Plus(vecV).Invert()
		vecD := currentCentere.Minus(u.Position)

		failDistance := distantion(u.Position, st.game.Zone.CurrentCenter)
		if failDistance > st.game.Zone.CurrentRadius*0.99 || (toCenter && toCenterTick > st.game.CurrentTick) {
			vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
			prj, prjOk := st.NearestProj(u)
			prjPt := prjectilePointPjr(u, prj)
			if prjOk {
				if u.OnPoint(prjPt, st.URadius()*2) {
					vecV1 := rotatePoints(prjPt, u.Position, 180.0)
					vecV = vecV1.Minus(vecV).Mult(MaxFU()) //по тапкам...
					l.Debug().Str("vecV", vecV.Log()).Str("prjPt", prjPt.Log()).Msg("prj on unit")
					failDistance := distantion(vecV, st.game.Zone.CurrentCenter)
					if failDistance > st.game.Zone.CurrentRadius*0.99 {
						vecV1 := rotatePoints(vecV, prjPt, 180.0)
						vecV = vecV1.Minus(vecV).Mult(MaxFU()) //по тапкам...
					}
				}
			}

			if toCenterTick == st.game.CurrentTick {
				toCenter = false
			} else {
				toCenterTick = st.game.CurrentTick + 10
				toCenter = true
			}
			st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
			l.Debug().Str("vecV", vecV.Log()).Str("prjPt", prjPt.Log()).Msg("walking to center")
			return
		}

		prj, prjOk := st.NearestProj(u)
		aim, aimOk := st.NearestAim(u)
		prop := st.consts.Weapons[u.WeaponIndex()]
		ammoD := prop.ProjectileSpeed / prop.ProjectileLifeTime
		if aim.WeaponIndex() == 2 {
		} else if d := u.Position.Distance(aim.Position); aimOk && d < ammoD {
			pv := nearestPoint(u.Position, aim.Position)
			vecV := pv.Noramalize().Mult(-1 * MaxFU())
			vecV = u.Position.Plus(vecV).Invert()
			vecD := aim.Position.Minus(u.Position)

			// идем по умолчанию от цели
			// 90% от дистанции выстрела и не под прицелом
			cause := ""
			if d <= ammoD*deltaDist {
				vecV = u.Position.Minus(aim.Position).Mult(MaxBU())
			}
			if prjOk {
				prjPt := prj.Position
				// если по нам прилет
				if u.OnPoint(prjPt, st.URadius()*2) {
					vecV1 := rotatePoints(prjPt, u.Position, 180.0)
					vecV = vecV1.Minus(u.Position).Mult(MaxFU()) //по тапкам...
				}
			}
			busy := LineAttackBussy(u, aim)
			act := NewActionOrderAim(!busy)
			action = &act
			if busy {
				loot, ok := st.NearestLootSheild(u)
				if ok && u.OnPoint(loot.Position, st.URadius()) {
					act := NewActionOrderPickup(loot.Id)
					action = &act
					vecV = loot.Position.Minus(u.Position)
					st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
					l.Log().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("moveAttackPickupLoot")
					return
				}

			}
			st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))

			l.Log().Str("cause", cause).Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("moveAttack")
			return
		} else if aimOk {

			if loot, ok := st.NearestLootWeapon(u); ok {
				wp := loot.Item.(ItemWeapon)
				d := distantion(loot.Position, u.Position)
				points := u.Ammo[wp.TypeIndex]
				prop := st.consts.Weapons[wp.TypeIndex]
				if d < 1.5*st.URadius() && wp.TypeIndex > u.WeaponIndex() && points < prop.MaxInventoryAmmo {
					vecV := loot.Position.Minus(u.Position).Mult(st.consts.MaxUnitForwardSpeed)
					vecD = loot.Position.Minus(u.Position)
					if ok && u.OnPoint(loot.Position, st.URadius()) {
						act := NewActionOrderPickup(loot.Id)
						action = &act
						st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
						st.usedWeapon[loot.Id] = struct{}{}

						l.Debug().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("pickupWP")
						return
					}
					st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
					l.Debug().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("pickupMoveWP")
					return
				}
			}

			cause := "ammo"
			vecV := aim.Position.Minus(u.Position)
			vecD = aim.Position.Minus(u.Position)

			loot, ok := st.NearestLootSheild(u)
			if ok && u.OnPoint(loot.Position, st.URadius()) {
				act := NewActionOrderPickup(loot.Id)
				action = &act
				vecV = loot.Position.Minus(u.Position)
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
				u.ActionResult = "moveAttackPickupLoot"
				l.Log().Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("moveAttackPickupLoot")
				return
			}

			if prjOk {
				prjPt := prj.Position
				// если по нам прилет
				if u.OnPoint(prjPt, st.URadius()*2) {
					cause = "under prject"
					vecV1 := rotatePoints(prjPt, u.Position, 180.0)
					vecV = vecV1.Minus(u.Position).Mult(MaxFU()) //по тапкам...
				}
			}

			if d := u.Position.Distance(aim.Position); d < ammoD { // можем стрелять и есть цель
				// 90% от дистанции выстрела и не под прицелом
				if d <= ammoD*deltaDist && !prjOk {
					vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
				}
			}
			// 90% от дистанции выстрела и под прицелом
			st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
			l.Log().Str("cause", cause).Str("vecV", vecV.Log()).Str("vecD", vecD.Log()).Msg("moveAttackToAim")
			return
		}

		{
			var vecV Vec2
			// prjOk := len(prjs) > 0
			if prjOk {
				prjPt := prj.Position
				// если по нам прилет
				if u.OnPoint(prjPt, st.URadius()*2) {
					vecV1 := rotatePoints(prjPt, u.Position, 180.0)
					vecV = vecV1.Minus(u.Position).Mult(MaxFU()) //по тапкам...
				}
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
				l.Debug().Str("v", vecV.Log()).Str("d", vecD.Log()).Msg("prj")
				return
			}
		}

		{
			pv := nearestPoint(u.Position, currentCentere)
			vecV := pv.Noramalize().Mult(-1 * MaxFU())
			vecV = u.Position.Plus(vecV).Invert()

			vecD := currentCentere.Minus(u.Position)

			if VelocityBusy(u) {
				vecD = rotate(u.Direction, math.Pi/6.0)
			}
			// if u.Velocity.Magnitude() < 0.01 {
			vecD = rotate(u.Direction, math.Pi/6.0)
			// }

			st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
			l.Debug().Str("v", vecV.Log()).Str("d", vecD.Log()).Msg("send")
		}
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
