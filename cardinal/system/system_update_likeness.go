package system

import (
	"fmt"

	comp "github.com/argus-labs/starter-game-template/cardinal/component"
	"github.com/argus-labs/starter-game-template/cardinal/tx"
	"pkg.world.dev/world-engine/cardinal/ecs"
	"pkg.world.dev/world-engine/cardinal/ecs/filter"
	"pkg.world.dev/world-engine/cardinal/ecs/storage"
)

// UpdateLikenessSystem is a system that updates the likeness value for a specific player and NPC.
func UpdateLikenessSystem(world *ecs.World, tq *ecs.TransactionQueue, _ *ecs.Logger) error {
	// Get all the transactions that are of type UpdateLikeness from the tx queue
	updateTxs := tx.UpdateLikeness.In(tq)

	// Create an index of player nicknames to their entity IDs
	playerNicknameToID := map[string]storage.EntityID{}
	ecs.NewQuery(filter.Exact(comp.Player, comp.Likeness)).Each(world, func(id storage.EntityID) bool {
		player, err := comp.Player.Get(world, id)
		if err != nil {
			return true
		}

		playerNicknameToID[player.Nickname] = id
		return true
	})

	// Iterate through all transactions and process them individually
	for _, update := range updateTxs {
		playerID, ok := playerNicknameToID[update.Value.PlayerNickname]
		// If the player doesn't exist, skip this transaction
		if !ok {
			tx.UpdateLikeness.AddError(world, update.TxHash,
				fmt.Errorf("player %q does not exist", update.Value.PlayerNickname))
			continue
		}

		// Get the likeness component for the player
		likeness, err := comp.Likeness.Get(world, playerID)
		if err != nil {
			tx.UpdateLikeness.AddError(world, update.TxHash,
				fmt.Errorf("can't get likeness for %q: %w", update.Value.PlayerNickname, err))
			continue
		}

		// Update the likeness value for the specific NPC
		likeness.Values[update.Value.NPCNickname] = update.Value.Value
		if err := comp.Likeness.Set(world, playerID, likeness); err != nil {
			tx.UpdateLikeness.AddError(world, update.TxHash,
				fmt.Errorf("failed to set likeness on %q: %w", update.Value.PlayerNickname, err))
			continue
		}

		// Indicate that the transaction was successful
		tx.UpdateLikeness.SetResult(world, update.TxHash, tx.UpdateLikenessReply{true})
	}

	return nil
}
