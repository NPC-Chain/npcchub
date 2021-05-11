package bank

import (
	"github.com/NPC-Chain/npcchub/codec"
)

// Register concrete types on codec codec
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterConcrete(MsgSend{}, "irishub/bank/Send", nil)
	cdc.RegisterConcrete(MsgIssue{}, "irishub/bank/Issue", nil)
	cdc.RegisterConcrete(MsgBurn{}, "irishub/bank/Burn", nil)
}

var msgCdc = codec.New()

func init() {
	RegisterCodec(msgCdc)
}
