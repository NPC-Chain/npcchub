package mint

import (
	"github.com/NPC-Chain/npcchub/codec"
)

// Register concrete types on codec codec
func RegisterCodec(cdc *codec.Codec) {
	// Not Register mint codec in app, deprecated now
	//cdc.RegisterConcrete(Minter{}, "irishub/mint/Minter", nil)
	cdc.RegisterConcrete(&Params{}, "irishub/mint/Params", nil)
}

var msgCdc = codec.New()

func init() {
	RegisterCodec(msgCdc)
}
