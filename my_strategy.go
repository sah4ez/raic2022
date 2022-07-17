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

	lShield         *sync.RWMutex
	nextSheildTicks map[int32]int32

	lAll   *sync.RWMutex
	allIDs map[int32]struct{}

	debugLock *sync.RWMutex

	l          *sync.RWMutex
	unitOrders map[int32]UnitOrder
	prevLoot   map[int32]Loot
	loot       map[int32]Loot

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

	var out io.Writer = os.Stdout
	if true {
		out = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	log.Logger = zerolog.New(out).Level(zerolog.DebugLevel).With().Timestamp().Logger()
	// if cfg.ReportCaller {
	// inlog = inlog.Caller()
	// }
	// if cfg.LogStack {
	// inlog = inlog.Stack()
	// }

	return &MyStrategy{
		units:      make(map[int32]Unit),
		prevUnits:  make(map[int32]Unit),
		consts:     constants,
		l:          new(sync.RWMutex),
		lAll:       new(sync.RWMutex),
		debugLock:  new(sync.RWMutex),
		unitOrders: make(map[int32]UnitOrder),
		prevLoot:   make(map[int32]Loot),
		loot:       make(map[int32]Loot),
		usedWeapon: make(map[int32]struct{}),
		unitRotate: make(map[int32]int32),

		lShield:         new(sync.RWMutex),
		nextSheildTicks: make(map[int32]int32),

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

	st.lAll.RLock()
	defer st.lAll.RUnlock()
	for k, v := range st.unitOrders {
		if _, ok := st.allIDs[k]; ok {
			unitOrders[k] = v
		}
	}

	return unitOrders
}

func (st MyStrategy) getOrder(game Game, debugInterface *DebugInterface) Order {
	st.Reset()

	st.lAll.Lock()
	st.allIDs = make(map[int32]struct{})
	st.lAll.Unlock()

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
	st.DoActionUnits()
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

		st.MoveUnit(log.Logger, "test action", u, st.NewUnitOrder(u, vecV, vecD, nil))
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
	st.lAll.Lock()
	defer st.lAll.Unlock()
	st.units = make(map[int32]Unit)
	st.aims = make([]Unit, 0)
	st.respAims = make([]Unit, 0)
	for _, u := range units {
		st.allIDs[u.Id] = struct{}{}
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
		// log.Debug().Str("p", s.Position.Log()).Msg("sound")
		st.sounds = append(st.sounds, s)
		// }
	}
}

func (st *MyStrategy) LoadLoot() {
	st.lAll.Lock()
	defer st.lAll.Unlock()
	for _, u := range st.game.Loot {
		st.allIDs[u.Id] = struct{}{}
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
			// fmt.Println("o>", o.Position, dAO, dUO, o.Radius, d)
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

func (st *MyStrategy) pickupWeapon(l zerolog.Logger, vecV Vec2, vecD Vec2, u Unit) bool {
	var action ActionOrder
	if u.IsArcher() {
		return false
	}
	loot, ok := st.NearestLootWeapon(u)
	if ok {
		if u.Weapon != nil {
			w, ok := loot.Item.(ItemWeapon)
			if !ok || u.Ammo[w.TypeIndex] == 0 {
				return false
			}
		}

		l = l.With().
			Str("loot", loot.Position.Log()).
			Logger()

		vecD = loot.Position.Minus(u.Position)
		vecV1 := loot.Position.Minus(u.Position)
		vecV1 = vecV1.Noramalize()
		vecV = vecV1.Mult(MaxFU())

		if ok && u.OnPoint(loot.Position, st.URadius()) {
			act := NewActionOrderPickup(loot.Id)
			action = &act
			st.MoveUnit(l, "pickup", u, st.NewUnitOrder(u, vecV, vecD, &action))
			return true
		}
		st.MoveUnit(l, "pickupMove", u, st.NewUnitOrder(u, vecV, vecD, nil))
		return true
	}
	return false
}

func (st *MyStrategy) pickupAmmo(l zerolog.Logger, vecV Vec2, vecD Vec2, u Unit) bool {
	loot, ok := st.NearestLootAmmo(u)
	if !ok {
		return false
	}

	a, ok := loot.Item.(ItemAmmo)
	if !ok {
		return false
	}
	idx := a.WeaponTypeIndex

	prop := st.consts.Weapons[idx]
	if u.Ammo[idx] == prop.MaxInventoryAmmo {
		return false
	}

	var action ActionOrder
	if ok && u.OnPoint(loot.Position, st.URadius()) {
		vecD = loot.Position.Minus(u.Position)
		vecV1 := loot.Position.Minus(u.Position)
		vecV1 = vecV1.Noramalize()
		vecV = vecV1.Mult(MaxFU())

		act := NewActionOrderPickup(loot.Id)
		action = &act
		st.MoveUnit(l, "pickupAmmo", u, st.NewUnitOrder(u, vecV, vecD, &action))
		return true
	} else if ok && u.OnPoint(loot.Position, st.URadius()*2.0) {
		vecD = loot.Position.Minus(u.Position)
		vecV1 := loot.Position.Minus(u.Position)
		vecV1 = vecV1.Noramalize()
		vecV = vecV1.Mult(MaxFU())

		st.MoveUnit(l, "pickupWalkingAmmo", u, st.NewUnitOrder(u, vecV, vecD, nil))
		return true
	}

	return false
}

func (st *MyStrategy) pickupWalkingAmmo(l zerolog.Logger, vecV Vec2, vecD Vec2, u Unit) bool {
	prop := st.consts.Weapons[u.WeaponIndex()]

	{
		full := false
		for i, prop := range st.consts.Weapons {
			if u.Ammo[i] == prop.MaxInventoryAmmo {
				full = full && true
			}
		}
		if full {
			return false
		}
	}

	var action ActionOrder
	loot, ok := st.NearestLootAmmoByTypeIndex(u)
	if u.Ammo[u.WeaponIndex()] == prop.MaxInventoryAmmo {
		loot, ok = st.NearestLootAmmo(u)
	}
	if ok && u.OnPoint(loot.Position, st.URadius()) {
		vecD = loot.Position.Minus(u.Position)
		vecV1 := loot.Position.Minus(u.Position)
		vecV1 = vecV1.Noramalize()
		vecV = vecV1.Mult(MaxFU())

		act := NewActionOrderPickup(loot.Id)
		action = &act
		st.MoveUnit(l, "pickupWalkingAmmo", u, st.NewUnitOrder(u, vecV, vecD, &action))
		return true
	} else if ok {
		vecD = loot.Position.Minus(u.Position)
		vecV1 := loot.Position.Minus(u.Position)
		vecV1 = vecV1.Noramalize()
		vecV = vecV1.Mult(MaxFU())

		st.MoveUnit(l, "pickupWalkingWAmmo", u, st.NewUnitOrder(u, vecV, vecD, nil))
		return true
	}

	return false
}

func (st *MyStrategy) pickupNoAmmo(l zerolog.Logger, vecV Vec2, vecD Vec2, u Unit) bool {
	var action ActionOrder

	loot, ok := st.NearestLootAmmoByTypeIndex(u)
	if !ok {
		return false
	}

	vecD = loot.Position.Minus(u.Position)
	vecV1 := loot.Position.Minus(u.Position)
	vecV1 = vecV1.Noramalize()
	vecV = vecV1.Mult(MaxFU())

	act := NewActionOrderPickup(loot.Id)
	action = &act
	st.MoveUnit(l, "pickupNoAmmo", u, st.NewUnitOrder(u, vecV, vecD, &action))
	return true
}

func (st *MyStrategy) pickupWalkingShield(l zerolog.Logger, vecV Vec2, vecD Vec2, u Unit) bool {
	var action ActionOrder

	if u.ShieldPotions == st.consts.MaxShieldPotionsInInventory {
		return false
	}

	loot, sheildOk := st.NearestLootSheild(u)
	if sheildOk && u.OnPoint(loot.Position, st.URadius()) {
		act := NewActionOrderPickup(loot.Id)
		action = &act
		vecV = loot.Position.Minus(u.Position)
		st.MoveUnit(l, "pickupWalkingShield", u, st.NewUnitOrder(u, vecV, vecD, &action))
		return true
	} else if sheildOk {
		vecV = loot.Position.Minus(u.Position)
		vecD = loot.Position.Minus(u.Position)
		vecV = vecV.Mult(st.consts.MaxUnitForwardSpeed)
		st.MoveUnit(l, "pickupWalkingShieldMove", u, st.NewUnitOrder(u, vecV, vecD, nil))
		return true
	}
	return false
}

func (st *MyStrategy) DoActionUnits() {
	wg := new(sync.WaitGroup)
	for _, u := range st.units {
		if u.Health == 0.0 {
			continue
		}
		wg.Add(1)
		go st.DoActionUnit(u, wg)
	}
	wg.Wait()
}

func (st *MyStrategy) FailDistance(l zerolog.Logger, u Unit, vecV Vec2, vecD Vec2) bool {

	failDistance := distantion(u.Position, st.game.Zone.CurrentCenter)
	if failDistance > st.game.Zone.CurrentRadius*0.98 {
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

		st.MoveUnit(l, "walking to center", u, st.NewUnitOrder(u, vecV, vecD, nil))
		return true
	}
	return false
}

func (st *MyStrategy) Respawn(l zerolog.Logger, u Unit, vecV Vec2, vecD Vec2) bool {

	if u.RemainingSpawnTime != nil && *u.RemainingSpawnTime > 0.0 {
		_, ok := st.NearestAim(u)
		if ok {
			for _, a := range st.aims {
				ap := a.Position.Noramalize()
				vecV = vecV.Plus(ap)
			}
			le := float64(len(st.aims))
			vecV = Vec2{vecV.X / le, vecV.Y / le}
			vecV = vecV.Mult(MaxFU()).Invert()
			st.MoveUnit(l, "spawn moving", u, st.NewUnitOrder(u, vecV, vecD, nil))
			return true
		}
		uu, uok := st.NearestUnit(u)
		if uok {
			vecV = u.Position.Minus(uu.Position)
			vecD = u.Position.Minus(uu.Position)
			st.MoveUnit(l, "spawn moving", u, st.NewUnitOrder(u, vecV, vecD, nil))
		}
	}
	return false
}

func (st *MyStrategy) Shield(l zerolog.Logger, u Unit, vecV Vec2, vecD Vec2, prjOk bool) bool {

	var action ActionOrder
	st.lShield.Lock()
	defer st.lShield.Unlock()

	nextSheildTick := st.nextSheildTicks[u.Id]

	loot, sheildOk := st.NearestLootSheild(u)
	if u.Shield < st.consts.MaxShield && nextSheildTick < st.game.CurrentTick {

		if u.ShieldPotions == st.consts.MaxShieldPotionsInInventory {
			l.Debug().Msg("full shield")
			return false
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
			st.nextSheildTicks[u.Id] = nextSheildTick
			vecD = rotate(u.Position, math.Pi)
			st.MoveUnit(l, "useSheild", u, st.NewUnitOrder(u, vecV, vecD, &action))
			return true
		} else if !prjOk {
			if sheildOk && u.OnPoint(loot.Position, st.URadius()) {
				act := NewActionOrderPickup(loot.Id)
				action = &act
				vecV = loot.Position.Minus(u.Position)
				st.MoveUnit(l, "pickupShield", u, st.NewUnitOrder(u, vecV, vecD, &action))
				return true
			} else if sheildOk {
				vecV = loot.Position.Minus(u.Position)
				vecD = loot.Position.Minus(u.Position)
				vecV = vecV.Mult(st.consts.MaxUnitForwardSpeed)
				st.MoveUnit(l, "pickupShieldMove", u, st.NewUnitOrder(u, vecV, vecD, nil))
				return true
			}
		}
	}
	return false
}

func (st *MyStrategy) DoActionUnit(u Unit, wg *sync.WaitGroup) {
	defer wg.Done()

	l := log.With().
		Int32("id", u.Id).
		Int32("tick", st.game.CurrentTick).
		Int32("shotTick", u.NextShotTick).
		Str("pos", u.Position.Log()).
		Str("st_v", u.Velocity.Log()).
		Str("st_d", u.Direction.Log()).
		Logger()

	var action ActionOrder

	prop := st.consts.Weapons[u.WeaponIndex()]
	vecV := zeroVec
	vecD := zeroVec

	if st.FailDistance(l, u, vecV, vecD) {
		return
	}

	if st.Respawn(l, u, vecV, vecD) {
		return
	}
	if u.Weapon == nil {
		if st.pickupWeapon(l, vecV, vecD, u) {
			return
		}
	}

	sound, soundOk := st.NearestSound(u)
	soundD := distantion(u.Position, sound.Position)
	soundProp := st.consts.Sounds[sound.TypeIndex]

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
	}

	if st.Shield(l, u, vecV, vecD, prjOk) {
		return
	}

	aim, aimOk := st.NearestAim(u)
	ammoD := prop.ProjectileSpeed / prop.ProjectileLifeTime
	if u.Ammo[u.WeaponIndex()] == 0 {
		l.Debug().Msg("no ammo... ahhh...!!!!")
		if st.pickupNoAmmo(l, vecV, vecD, u) {
			return
		}
	} else if d := u.Position.Distance(aim.Position); aimOk && d < ammoD && u.CanShoot(st.game.CurrentTick) {
		// направляем вектор в позицию относительно скорости
		vecD = aim.Position.Minus(u.Position) //.Plus(aim.Direction.Scalar(aim.Velocity))

		vecV = aim.Position.Minus(u.Position)
		if u.IsArcher() {
			vecV = aim.Position.Plus(u.Position)
		}
		// идем по умолчанию от цели
		// 90% от дистанции выстрела и не под прицелом
		if d <= ammoD*deltaDist && !prjOk {
			// vecV = vecV.Mult(-1.0)
			// пытаемся сместиться в случае отступления к центру карты
			vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
			// 90% от дистанции выстрела и под прицелом
		} else if d <= ammoD*deltaDist && prjOk {
			// лучник и второй тип
			vecV = u.Position.Minus(aim.Position).Mult(MaxBU()) // пытаемся максимально отойти от этих ...
		}
		if prjPoint != nil {
			prjV, ll := checkProject(l, *prjPoint, u, aim)
			if !prjV.IsZero() {
				vecV = prjV
				l = ll
			}
		}
		if soundOk && soundD <= soundProp.Distance {
			l.Debug().Str("soudnName", soundProp.Name).Msg("under fire sound")
			if soundProp.Name == "BowHit" {
				vecD = sound.Position
			}
			if soundProp.Name == "Staff" || soundProp.Name == "StaffHit" {
				vecV1 := sound.Position.Noramalize()
				vecV1 = vecV1.Mult(MaxBU())
				vecV = u.Position.Plus(vecV1)
			}
		}
		busy := LineAttackBussy(u, aim)
		l = l.With().Float64("aim", u.Aim).Bool("busy", busy).Logger()

		act := NewActionOrderAim(!busy)
		action = &act
		if busy {
			if !u.IsArcher() {
				if st.pickupWeapon(l, vecV, vecD, u) {
					return
				} else if st.pickupAmmo(l, vecV, vecD, u) {
					return
				}
			}
		}
		// if aim.WeaponIndex() == 2 {
		// vecV2 := st.game.Zone.CurrentCenter.Minus(vecV)
		// vecV2 = vecV2.Noramalize()
		// vecV = vecV.Minus(vecV2.Mult(3.0))
		// }
		// if aim.WeaponIndex() == 1 {
		// vecV2 := st.game.Zone.CurrentCenter.Minus(vecV)
		// vecV2 = vecV2.Noramalize()
		// vecV = vecV.Minus(vecV2.Mult(3.0))
		// }
		st.MoveUnit(l, "moveAttack", u, st.NewUnitOrder(u, vecV, vecD, &action))
		return
	} else if aimOk {
		vecV = aim.Position.Minus(u.Position)
		vecD = aim.Position.Minus(u.Position)

		loot, ok := st.NearestLootSheild(u)
		if ok && u.OnPoint(loot.Position, st.URadius()) {
			act := NewActionOrderPickup(loot.Id)
			action = &act
			st.MoveUnit(l, "moveAttackPickupLoot", u, st.NewUnitOrder(u, vecV, vecD, &action))
			return
		}
		if soundOk && soundD <= soundProp.Distance {
			l.Debug().Str("soudnName", soundProp.Name).Msg("under fire sound")
			if soundProp.Name == "BowHit" {
				vecD = sound.Position
			}
			if soundProp.Name == "Staff" || soundProp.Name == "StaffHit" {
				vecV = sound.Position.Plus(u.Position)
				st.MoveUnit(l, "moveAttackToAim", u, st.NewUnitOrder(u, vecV, vecD, nil))
				return
			}
		}

		if prjPoint != nil {
			prjV, ll := checkProject(l, *prjPoint, u, aim)
			if !prjV.IsZero() {
				vecV = prjV
				l = ll
			}
		}

		if d := u.Position.Distance(aim.Position); d < ammoD { // можем стрелять и есть цель
			// 90% от дистанции выстрела и не под прицелом
			if d <= ammoD*deltaDist && !prjOk {
				// vecV = vecV.Mult(-1.0)
				// пытаемся сместиться в случае отступления к центру карты
				// vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
			} else if st.pickupWeapon(l, vecV, vecD, u) {
				return
			} else if st.pickupAmmo(l, vecV, vecD, u) {
				return
			}
		}
		st.MoveUnit(l, "moveAttackToAim", u, st.NewUnitOrder(u, vecV, vecD, nil))
		return
	}
	if st.pickupWeapon(l, vecV, vecD, u) {
		return
	}
	if st.pickupWalkingAmmo(l, vecV, vecD, u) {
		return
	}
	if st.pickupWalkingShield(l, vecV, vecD, u) {
		return
	}
	vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
	vecD = st.game.Zone.CurrentCenter.Minus(u.Position)

	// todo собирать лут
	vecD = rotate(u.Direction, 5.0)
	st.MoveUnit(l, "simpleWalking", u, st.NewUnitOrder(u, vecV, vecD, nil))
}

func (st *MyStrategy) NewUnitOrder(u Unit, v Vec2, d Vec2, a *ActionOrder) UnitOrder {

	uo := NewUnitOrder(v, d, a)
	return uo
}

func (st *MyStrategy) PrintLootInfo() {

	st.debugLock.Lock()
	defer st.debugLock.Unlock()

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

func (st *MyStrategy) MoveUnit(l zerolog.Logger, msg string, u Unit, o UnitOrder) {
	st.PrintUnitInfo(u)
	st.l.Lock()
	defer st.l.Unlock()

	l = newWalking(l, o.TargetVelocity, o.TargetDirection)
	l.Debug().Msg(msg)

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
