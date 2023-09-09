package system

import (
	"fmt"

	comp "github.com/argus-labs/starter-game-template/cardinal/component"
	"github.com/argus-labs/starter-game-template/cardinal/tx"
	"pkg.world.dev/world-engine/cardinal/ecs"
)

// PlayerSpawnerSystem is a system that spawns players based on `CreatePlayer` transactions.
func PlayerSpawnerSystem(world *ecs.World, tq *ecs.TransactionQueue, _ *ecs.Logger) error {
	// Get all the transactions that are of type CreatePlayer from the tx queue
	createTxs := tx.CreatePlayer.In(tq)

	// Iterate through all transactions and process them individually.
	for _, create := range createTxs {
		// Create a new entity with Player and Likeness components
		id, err := world.Create(comp.Player, comp.Likeness)
		if err != nil {
			tx.CreatePlayer.AddError(world, create.TxHash,
				fmt.Errorf("error creating player: %w", err))
			continue
		}

		// Set the Nickname field of the Player component
		err = comp.Player.Set(world, id, comp.PlayerComponent{Nickname: create.Value.Nickname})
		if err != nil {
			tx.CreatePlayer.AddError(world, create.TxHash,
				fmt.Errorf("error setting player nickname: %w", err))
			continue
		}

		// Initialize an empty map for the Likeness component
		err = comp.Likeness.Set(world, id, comp.LikenessComponent{Values: make(map[string]int)})
		if err != nil {
			tx.CreatePlayer.AddError(world, create.TxHash,
				fmt.Errorf("error setting player likeness: %w", err))
			continue
		}

		// Indicate that the transaction was successful
		tx.CreatePlayer.SetResult(world, create.TxHash, tx.CreatePlayerMsgReply{true})
	}

	return nil
}
