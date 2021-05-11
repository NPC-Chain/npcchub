package params

import (
	"github.com/NPC-Chain/npcchub/codec"
)

// Register concrete types on codec codec
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterInterface((*ParamSet)(nil), nil)
}
