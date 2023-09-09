package tx

import (
	"pkg.world.dev/world-engine/cardinal/ecs"
)

// UpdateLikenessMsg is the message struct for updating likeness.
type UpdateLikenessMsg struct {
	PlayerNickname string `json:"player_nickname"`
	NPCNickname    string `json:"npc_nickname"`
	Value          int    `json:"value"`
}

// UpdateLikenessReply is the reply struct for the update likeness transaction.
type UpdateLikenessReply struct {
	Success bool `json:"success"`
}

// UpdateLikeness is the transaction type for updating likeness.
var UpdateLikeness = ecs.NewTransactionType[UpdateLikenessMsg, UpdateLikenessReply]("update-likeness")
