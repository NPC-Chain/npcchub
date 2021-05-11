package lite

import (
	"github.com/NPC-Chain/npcchub/codec"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

var cdc = codec.New()

func init() {
	ctypes.RegisterAmino(cdc)
}
