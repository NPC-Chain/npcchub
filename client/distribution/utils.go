package distribution

import (
	"github.com/NPC-Chain/npcchub/app/v1/distribution"
	"github.com/NPC-Chain/npcchub/client/context"
	"github.com/NPC-Chain/npcchub/client/utils"
	sdk "github.com/NPC-Chain/npcchub/types"
)

// distribution info for a particular validator
type ValidatorDistInfoOutput struct {
	OperatorAddr            sdk.ValAddress          `json:"operator_addr"`
	FeePoolWithdrawalHeight int64                   `json:"fee_pool_withdrawal_height"`
	DelAccum                distribution.TotalAccum `json:"del_accum"`
	DelPool                 string                  `json:"del_pool"`
	ValCommission           string                  `json:"val_commission"`
}

func ConvertToValidatorDistInfoOutput(cliCtx context.CLIContext, vdi distribution.ValidatorDistInfo) ValidatorDistInfoOutput {
	exRate := utils.ExRateFromStakeTokenToMainUnit(cliCtx)
	delPool := utils.ConvertDecToRat(vdi.DelPool.AmountOf(sdk.IrisAtto)).Mul(exRate).FloatString() + sdk.Iris
	valCommission := utils.ConvertDecToRat(vdi.ValCommission.AmountOf(sdk.IrisAtto)).Mul(exRate).FloatString() + sdk.Iris
	return ValidatorDistInfoOutput{
		OperatorAddr:            vdi.OperatorAddr,
		FeePoolWithdrawalHeight: vdi.FeePoolWithdrawalHeight,
		DelAccum:                vdi.DelAccum,
		DelPool:                 delPool,
		ValCommission:           valCommission,
	}
}
