package main

import (
	. "ai_cup_22/debugging"
	. "ai_cup_22/model"
	"fmt"
	"sync"
)

var (
	white    = NewColor(255.0, 255.0, 255.0, 1.0)
	black    = NewColor(0.0, 0.0, 0.0, 1.0)
	red      = NewColor(150.0, 0.0, 0.0, 1.0)
	green    = NewColor(0.0, 150.0, 0.0, 1.0)
	blue     = NewColor(0.0, 0.0, 150.0, 1.0)
	zeroVec  = Vec2{0, 0}
	oneVec   = Vec2{1, 1}
	twoVec   = Vec2{2, 2}
	halfVec  = Vec2{0.5, 0.5}
	leftVec  = Vec2{0, 0.5}
	rightVec = Vec2{0, 0.5}
	mainSize = 12.0
)

type MyStrategy struct {
	debugInterface *DebugInterface
	units          map[int32]Unit
	consts         Constants
	game           Game

	l              *sync.RWMutex
	unitOrders     map[int32]UnitOrder
	prevUnitOrders map[int32]UnitOrder
}

func NewMyStrategy(constants Constants) *MyStrategy {
	return &MyStrategy{
		units:          make(map[int32]Unit),
		consts:         constants,
		l:              new(sync.RWMutex),
		unitOrders:     make(map[int32]UnitOrder),
		prevUnitOrders: make(map[int32]UnitOrder),
	}
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
	st.debugInterface = debugInterface
	st.game = game

	var me Player
	for _, p := range game.Players {
		if p.Id == game.MyId {
			me = p
			break
		}
	}
	for _, u := range game.Units {
		if u.PlayerId == me.Id {
			if _, ok := st.units[u.Id]; !ok {
				fmt.Println("added", u.Id)
			}
			st.units[u.Id] = u
		}
	}
	for _, u := range st.units {

		vecD := Vec2{
			-1.0 * (u.Position.X + u.Direction.X + 0.1),
			-1.0 * (u.Position.Y + u.Direction.Y + 0.1),
		}
		vecV := Vec2{
			-1.0 * (u.Position.X + u.Velocity.X + 0.1),
			-1.0 * (u.Position.Y + u.Velocity.Y + 0.1),
		}
		var order ActionOrder
		if u.Health < 50.0 {
			aim := NewActionOrderUseShieldPotion()
			order = &aim
		} else {
			aim := NewActionOrderAim(u.Aim > 0.8)
			order = &aim
		}
		cmd := NewUnitOrder(vecV, vecD, &order)
		st.MoveUnit(u, cmd)
	}

	return Order{
		UnitOrders: st.GetOrders(),
	}
}

func (st MyStrategy) MoveUnit(u Unit, o UnitOrder) {
	st.PrintUnitInfo(u)
	// if eo, ok := st.unitOrders[u.Id]; ok {
	// eo.TargetVelocity = o.TargetVelocity
	// eo.TargetDirection = o.TargetDirection
	// st.unitOrders[u.Id] = eo
	// } else {
	if _, ok := st.prevUnitOrders[u.Id]; !ok {
		st.prevUnitOrders[u.Id] = o
	} else {
		st.prevUnitOrders[u.Id] = st.unitOrders[u.Id]
	}
	st.unitOrders[u.Id] = o
	// }
}

func (st MyStrategy) debugUpdate(debugInterface DebugInterface) {
	// debugInterface.SetAutoFlush(true)
	// matrix := [][]Vec2{
	// []Vec2{{0, 0}, {0, 100}, {0, 200}, {0, 300}},
	// []Vec2{{100, 0}, {100, 100}, {100, 200}, {100, 300}},
	// []Vec2{{200, 0}, {200, 100}, {200, 200}, {200, 300}},
	// []Vec2{{300, 0}, {300, 100}, {300, 200}, {300, 300}},
	// }
	//
	// for _, x := range matrix {
	// for _, p := range x {
	// debugInterface.AddPlacedText(
	// p,
	// fmt.Sprintf("%.2f:%.2f", p.X, p.Y),
	// Vec2{1.0, 1.0},
	// mainSize,
	// black,
	// )
	// }
	// }
}

func (st MyStrategy) PrintUnitInfo(u Unit) {
	if st.debugInterface == nil {
		return
	}

	st.debugInterface.SetAutoFlush(true)

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
		fmt.Sprintf("units: %d", len(st.game.Units)),
		fmt.Sprintf(" loot: %d", len(st.game.Loot)),
	}
	for _, u := range st.units {
		info = append(info, fmt.Sprintf("%d weapon: %d", u.Id, u.WeaponIndex()))
		info = append(info, fmt.Sprintf("%d ammo: %d", u.Id, len(u.Ammo)))
		info = append(info, fmt.Sprintf("%d ns: %d", u.Id, u.NextShotTick))
		info = append(info, fmt.Sprintf("%d Aim: %.4f", u.Id, u.Aim))
	}
	for i, msg := range info {
		st.debugInterface.AddPlacedText(
			Vec2{-350, -350 + float64(i)*float64(mainSize+2.0)},
			msg,
			Vec2{1.0, 1.0},
			mainSize,
			black,
		)
	}
	for _, u := range st.units {
		st.debugInterface.AddSegment(
			u.Position,
			st.unitOrders[u.Id].TargetDirection,
			0.2,
			black,
		)
		st.debugInterface.AddSegment(
			u.Position,
			u.Velocity,
			0.2,
			blue,
		)
		st.debugInterface.AddSegment(
			u.Position,
			u.Direction,
			0.2,
			green,
		)
	}
}

func (strategy MyStrategy) finish() {}
