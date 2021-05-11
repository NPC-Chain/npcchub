package slashing

import (
	"fmt"
	"github.com/NPC-Chain/npcchub/mock/baseapp"
	"github.com/NPC-Chain/npcchub/mock/simulation"
	"github.com/NPC-Chain/npcchub/modules/slashing"
	sdk "github.com/NPC-Chain/npcchub/types"
	"math/rand"
)

// SimulateMsgUnjail
func SimulateMsgUnjail(k slashing.Keeper) simulation.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simulation.Account, event func(string)) (action string, fOp []simulation.FutureOperation, err error) {
		acc := simulation.RandomAcc(r, accs)
		address := sdk.ValAddress(acc.Address)
		msg := slashing.NewMsgUnjail(address)
		if msg.ValidateBasic() != nil {
			return "", nil, fmt.Errorf("expected msg to pass ValidateBasic: %s", msg.GetSignBytes())
		}
		ctx, write := ctx.CacheContext()
		result := slashing.NewHandler(k)(ctx, msg)
		if result.IsOK() {
			write()
		}
		event(fmt.Sprintf("slashing/MsgUnjail/%v", result.IsOK()))
		action = fmt.Sprintf("TestMsgUnjail: ok %v, msg %s", result.IsOK(), msg.GetSignBytes())
		return action, nil, nil
	}
}
