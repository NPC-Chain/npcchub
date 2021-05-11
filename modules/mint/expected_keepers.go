package mint

import (
	sdk "github.com/NPC-Chain/npcchub/types"
)

// expected fee collection keeper interface
type FeeKeeper interface {
	AddCollectedFees(sdk.Context, sdk.Coins) sdk.Coins
}
