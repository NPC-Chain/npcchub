package auth

import (
	"github.com/NPC-Chain/npcchub/codec"
)

// Register concrete types on codec codec for default AppAccount
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterInterface((*Account)(nil), nil)
	cdc.RegisterConcrete(&BaseAccount{}, "irishub/bank/Account", nil)
	cdc.RegisterConcrete(StdTx{}, "irishub/bank/StdTx", nil)
	cdc.RegisterConcrete(&Params{}, "irishub/Auth/Params", nil)
}

var msgCdc = codec.New()

func init() {
	RegisterCodec(msgCdc)
	codec.RegisterCrypto(msgCdc)
}
