package component

import "pkg.world.dev/world-engine/cardinal/ecs"

type LikenessComponent struct {
	Values map[string]int // Mapping from npc name to likeness value
}

var Likeness = ecs.NewComponentType[LikenessComponent]()
