package gov

import (
	sdk "github.com/NPC-Chain/npcchub/types"
)

// AllInvariants tests all governance invariants
func AllInvariants() sdk.Invariant {
	return func(ctx sdk.Context) error {
		// TODO Add some invariants!
		// Checking proposal queues, no passed-but-unexecuted proposals, etc.
		return nil
	}
}
