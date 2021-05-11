package slashing

import (
	sdk "github.com/NPC-Chain/npcchub/types"
)

// TODO Any invariants to check here?
// AllInvariants tests all slashing invariants
func AllInvariants() sdk.Invariant {
	return func(ctx sdk.Context) error {
		return nil
	}
}
