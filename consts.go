package main

import (
	. "ai_cup_22/debugging"
	. "ai_cup_22/model"
)

var (
	white          = NewColor(255.0, 255.0, 255.0, 1.0)
	black          = NewColor(0.0, 0.0, 0.0, 1.0)
	black05        = NewColor(0.0, 0.0, 0.0, 0.5)
	black25        = NewColor(0.0, 0.0, 0.0, 0.25)
	red            = NewColor(150.0, 0.0, 0.0, 1.0)
	red05          = NewColor(150.0, 0.0, 0.0, 0.5)
	red025         = NewColor(150.0, 0.0, 0.0, 0.25)
	green          = NewColor(0.0, 150.0, 0.0, 1.0)
	green05        = NewColor(0.0, 150.0, 0.0, 0.5)
	blue           = NewColor(0.0, 0.0, 150.0, 1.0)
	blue05         = NewColor(0.0, 0.0, 150.0, 0.5)
	blue25         = NewColor(0.0, 0.0, 150.0, 0.25)
	zeroVec        = Vec2{0, 0}
	oneVec         = Vec2{1, 1}
	twoVec         = Vec2{2, 2}
	halfVec        = Vec2{0.5, 0.5}
	leftVec        = Vec2{0, 0.5}
	rightVec       = Vec2{0, 0.5}
	twoSize        = 1.0
	mainSize       = 1.0
	lootSize       = 0.2
	lineSize       = 1.0
	prSize         = 0.1
	lineAttackSize = 1.0
	bigLineSize    = 0.5
	angleWalk      = 1.0
	deltaDist      = 0.85

	soundRingRadius = 1.2
	soundRingSize   = 0.2
)
