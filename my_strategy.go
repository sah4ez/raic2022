package main

import (
	. "ai_cup_22/debugging"
	. "ai_cup_22/model"
	"fmt"
	"math"
	"sort"
	"sync"
)

var (
	white          = NewColor(255.0, 255.0, 255.0, 1.0)
	black          = NewColor(0.0, 0.0, 0.0, 1.0)
	black05        = NewColor(0.0, 0.0, 0.0, 0.5)
	red            = NewColor(150.0, 0.0, 0.0, 1.0)
	red05          = NewColor(150.0, 0.0, 0.0, 0.5)
	red025         = NewColor(150.0, 0.0, 0.0, 0.25)
	green          = NewColor(0.0, 150.0, 0.0, 1.0)
	green05        = NewColor(0.0, 150.0, 0.0, 0.5)
	blue           = NewColor(0.0, 0.0, 150.0, 1.0)
	zeroVec        = Vec2{0, 0}
	oneVec         = Vec2{1, 1}
	twoVec         = Vec2{2, 2}
	halfVec        = Vec2{0.5, 0.5}
	leftVec        = Vec2{0, 0.5}
	rightVec       = Vec2{0, 0.5}
	mainSize       = 1.0
	lootSize       = 0.2
	lineSize       = 1.0
	lineAttackSize = 1.0
	bigLineSize    = 2.2
	angleWalk      = 1.0
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

	lootsW   []Loot
	lootsA   []Loot
	lootsS   []Loot
	aims     []Unit
	prevAims []Unit

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

		aims:     make([]Unit, 0),
		prevAims: make([]Unit, 0),

		haims:     make(map[int32]Unit),
		prevHaims: make(map[int32]Unit),

		once: new(sync.Once),
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
	st.LoadLoot()
	st.DoActionUnit()
	st.PrintLootInfo()

	return Order{
		UnitOrders: st.GetOrders(),
	}
}

func (st *MyStrategy) LoadUnits(units []Unit) {
	inFocus := false
	for _, u := range units {
		if u.PlayerId == st.me.Id {
			st.units[u.Id] = u
		} else {
			if st.aim != nil && st.aim.Id == u.Id {
				inFocus = true
			}
			st.aims = append(st.aims, u)
		}
	}
	if !inFocus {
		st.aim = nil
	}
	return
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

func (st *MyStrategy) DoActionUnit() {

	for _, u := range st.units {
		var action ActionOrder

		vecV := zeroVec
		vecD := zeroVec

		if u.Ammo[u.WeaponIndex()] < int32(float64(st.consts.Weapons[u.WeaponIndex()].MaxInventoryAmmo)*0.1) {
			loot, ok := st.NearestLootAmmo(u)

			if ok && u.Position.Distance(loot.Position) < st.consts.UnitRadius {
				act := NewActionOrderPickup(loot.Id)
				action = &act
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
				return
			} else if ok {
				vecV = loot.Position.Minus(u.Position)
				vecD = loot.Position.Minus(u.Position)
			}
		}

		if u.Health < float64(st.consts.UnitHealth/0.5) {
			if u.ShieldPotions > 0 {
				act := NewActionOrderUseShieldPotion()
				action = &act
				st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
				return
			} else {
				loot, ok := st.NearestLootSheild(u)
				if ok {
					if ok && u.Position.Distance(loot.Position) < st.consts.UnitRadius {
						act := NewActionOrderPickup(loot.Id)
						action = &act
						st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
						return
					} else if ok {
						vecV = loot.Position.Minus(u.Position)
						vecD = loot.Position.Minus(u.Position)
					}
				}
			}
		}

		aim, ok := st.NearestAim(u)

		prop := st.consts.Weapons[u.WeaponIndex()]
		ammoD := prop.ProjectileSpeed / prop.ProjectileLifeTime
		if d := u.Position.Distance(aim.Position); ok && d < ammoD {
			delta := 0.9
			vecD = aim.Position.Minus(u.Position)
			vecV = aim.Position.Minus(u.Position)
			fmt.Println(">>>", vecV, prop.ProjectileSpeed, prop.ProjectileLifeTime, d, ammoD)
			if d <= ammoD*delta {
				vecV = vecV.Mult(-1.0)
			}
			act := NewActionOrderAim(true)
			action = &act
			st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, &action))
			return
		} else if ok {
			vecV = aim.Position.Minus(u.Position)
			vecD = aim.Position.Minus(u.Position)
		} else {
			vecV = st.game.Zone.CurrentCenter.Minus(u.Position)
			vecD = st.game.Zone.CurrentCenter.Minus(u.Position)
		}

		st.MoveUnit(u, st.NewUnitOrder(u, vecV, vecD, nil))
	}
}

func (st *MyStrategy) Attack(u Unit) UnitOrder {
	if st.aim == nil {
		sort.Sort(NewByDistance(u, st.aims))
		st.aim = &st.aims[0]
	}

	var order ActionOrder
	aim := NewActionOrderAim(u.Aim > 0.1)
	order = &aim

	angle := math.Atan2(st.aim.Position.Y, st.aim.Position.X) - math.Atan2(u.Position.Y, u.Position.X)
	vecD := rotate(u.Position, angle)
	// vecD := st.aim.Position
	vecV := st.aim.Position

	return st.NewUnitOrder(u, vecV, vecD, &order)
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

	debugInterface.AddCircle(st.game.Zone.CurrentCenter, 5, red)

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
		fmt.Sprintf(" tick: %d", st.game.CurrentTick),
		fmt.Sprintf("units: %d", len(st.aims)),
		fmt.Sprintf(" loots: a:%d  w:%d  s:%d", len(st.lootsA), len(st.lootsW), len(st.lootsS)),
	}
	for _, u := range st.units {
		info = append(info, fmt.Sprintf("p: %.2f : %.2f", u.Position.X, u.Position.Y))
		info = append(info, fmt.Sprintf("v: %.2f : %.2f", u.Velocity.X, u.Velocity.Y))
		info = append(info, fmt.Sprintf("d: %.2f : %.2f", u.Direction.X, u.Direction.Y))

		info = append(info, fmt.Sprintf("%d weapon: %d", u.Id, u.WeaponIndex()))
		info = append(info, fmt.Sprintf("%d ammo: %d", u.Id, len(u.Ammo)))
		info = append(info, fmt.Sprintf("%d ns: %d", u.Id, u.NextShotTick))
		info = append(info, fmt.Sprintf("%d Aim: %.4f", u.Id, u.Aim))
		info = append(info, fmt.Sprintf("%d Aim: %.4f", u.Id, u.ShieldPotions))
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
	for _, u := range st.units {
		// fmt.Println(
		// ">>>>>",
		// u.Position,
		// u.Direction,
		// st.unitOrders[u.Id].TargetDirection,
		// st.unitOrders[u.Id].TargetVelocity,
		// u.Velocity,
		// )
		// st.debugInterface.AddSegment(
		// u.Position,
		// st.unitOrders[u.Id].TargetDirection,
		// lineSize,
		// black,
		// )
		st.debugInterface.AddSegment(
			u.Position,
			Vec2{u.Position.X + u.Velocity.X*u.Direction.X, u.Position.Y + u.Velocity.Y*u.Direction.Y},
			lineSize,
			black05,
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
		if st.aim != nil {
			st.debugInterface.AddSegment(
				u.Position,
				st.aim.Position,
				lineSize,
				green,
			)
		}
	}
}

func (st *MyStrategy) finish() {

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
