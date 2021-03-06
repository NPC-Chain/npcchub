package types

import (
	"fmt"
	"github.com/tendermint/tendermint/libs/log"

	sdk "github.com/NPC-Chain/npcchub/types"
)

// common parameters used in withdraws from validators
type WithdrawContext struct {
	FeePool        FeePool
	Height         int64   // block height
	TotalPower     sdk.Dec // total bonded tokens in the network
	ValPower       sdk.Dec // validator's bonded tokens
	CommissionRate sdk.Dec // validator commission rate
}

func NewWithdrawContext(feePool FeePool, height int64, totalPower,
	valPower, commissionRate sdk.Dec) WithdrawContext {

	return WithdrawContext{
		FeePool:        feePool,
		Height:         height,
		TotalPower:     totalPower,
		ValPower:       valPower,
		CommissionRate: commissionRate,
	}
}

//_____________________________________________________________________________

// distribution info for a particular validator
type ValidatorDistInfo struct {
	OperatorAddr sdk.ValAddress `json:"operator_addr"`

	FeePoolWithdrawalHeight int64 `json:"fee_pool_withdrawal_height"` // last height this validator withdrew from the global pool

	DelAccum      TotalAccum `json:"del_accum"`      // total accumulation factor held by delegators
	DelPool       DecCoins   `json:"del_pool"`       // rewards owed to delegators, commission has already been charged (includes proposer reward)
	ValCommission DecCoins   `json:"val_commission"` // commission collected by this validator (pending withdrawal)
}

func NewValidatorDistInfo(operatorAddr sdk.ValAddress, currentHeight int64) ValidatorDistInfo {
	return ValidatorDistInfo{
		OperatorAddr:            operatorAddr,
		FeePoolWithdrawalHeight: currentHeight,
		DelPool:                 DecCoins{},
		DelAccum:                NewTotalAccum(currentHeight),
		ValCommission:           DecCoins{},
	}
}

// update total delegator accumululation
func (vi ValidatorDistInfo) UpdateTotalDelAccum(logger log.Logger, height int64, totalDelShares sdk.Dec) ValidatorDistInfo {
	vi.DelAccum = vi.DelAccum.UpdateForNewHeight(logger, height, totalDelShares)
	return vi
}

// Get the total delegator accum within this validator at the provided height
func (vi ValidatorDistInfo) GetTotalDelAccum(height int64, totalDelShares sdk.Dec) sdk.Dec {
	return vi.DelAccum.GetAccum(height, totalDelShares)
}

// Get the validator accum at the provided height
func (vi ValidatorDistInfo) GetValAccum(height int64, valTokens sdk.Dec) sdk.Dec {
	blocks := height - vi.FeePoolWithdrawalHeight
	return valTokens.MulInt(sdk.NewInt(blocks))
}

// Move any available accumulated fees in the FeePool to the validator's pool
// - updates validator info's FeePoolWithdrawalHeight, thus setting accum to 0
// - updates fee pool to latest height and total val accum w/ given totalBonded
// This is the only way to update the FeePool's validator TotalAccum.
// NOTE: This algorithm works as long as TakeFeePoolRewards is called after every power change.
// - called in ValidationDistInfo.WithdrawCommission
// - called in DelegationDistInfo.WithdrawRewards
// NOTE: When a delegator unbonds, say, onDelegationSharesModified ->
//       WithdrawDelegationReward -> WithdrawRewards
func (vi ValidatorDistInfo) TakeFeePoolRewards(logger log.Logger, wc WithdrawContext) (
	ValidatorDistInfo, FeePool) {

	fp := wc.FeePool.UpdateTotalValAccum(logger, wc.Height, wc.TotalPower)

	if fp.TotalValAccum.Accum.IsZero() {
		logger.Debug("The total validator accumulation of feelPool is zero")
		vi.FeePoolWithdrawalHeight = wc.Height
		return vi, fp
	}

	// update the validators pool
	accum := vi.GetValAccum(wc.Height, wc.ValPower)
	vi.FeePoolWithdrawalHeight = wc.Height

	if accum.GT(fp.TotalValAccum.Accum) {
		panic("individual accum should never be greater than the total")
	}
	withdrawalTokens := fp.ValPool.MulDec(accum).QuoDec(fp.TotalValAccum.Accum)
	remValPool := fp.ValPool.Minus(withdrawalTokens)

	commission := withdrawalTokens.MulDec(wc.CommissionRate)
	afterCommission := withdrawalTokens.Minus(commission)

	fp.TotalValAccum.Accum = fp.TotalValAccum.Accum.Sub(accum)
	fp.ValPool = remValPool
	logger.Debug("Global fee pool", "total_accum", fp.TotalValAccum.Accum.String(), "update_height", fp.TotalValAccum.UpdateHeight, "val_pool", fp.ValPool.ToString(), "community_pool", fp.CommunityPool.ToString())
	vi.ValCommission = vi.ValCommission.Plus(commission)
	vi.DelPool = vi.DelPool.Plus(afterCommission)
	logger.Debug("Take fee pool reward", "withdraw_token", withdrawalTokens.ToString(), "commission_rate", wc.CommissionRate.String(), "commission", commission.ToString(), "after_commission", afterCommission.ToString())

	return vi, fp
}

// withdraw commission rewards
func (vi ValidatorDistInfo) WithdrawCommission(logger log.Logger, wc WithdrawContext) (
	vio ValidatorDistInfo, fpo FeePool, withdrawn DecCoins) {

	vi, fp := vi.TakeFeePoolRewards(logger, wc)
	logger.Debug("Withdraw commission", "commission", vi.ValCommission.ToString())
	withdrawalTokens := vi.ValCommission
	vi.ValCommission = DecCoins{} // zero

	return vi, fp, withdrawalTokens
}

// get the validator's pool rewards at this current state,
func (vi ValidatorDistInfo) CurrentPoolRewards(
	wc WithdrawContext) DecCoins {

	fp := wc.FeePool
	totalValAccum := fp.GetTotalValAccum(wc.Height, wc.TotalPower)
	valAccum := vi.GetValAccum(wc.Height, wc.ValPower)

	if valAccum.GT(totalValAccum) {
		panic("individual accum should never be greater than the total")
	}
	withdrawalTokens := fp.ValPool.MulDec(valAccum).QuoDec(totalValAccum)
	commission := withdrawalTokens.MulDec(wc.CommissionRate)
	afterCommission := withdrawalTokens.Minus(commission)
	pool := vi.DelPool.Plus(afterCommission)
	return pool
}

// get the validator's commission pool rewards at this current state,
func (vi ValidatorDistInfo) CurrentCommissionRewards(
	wc WithdrawContext) DecCoins {

	fp := wc.FeePool
	totalValAccum := fp.GetTotalValAccum(wc.Height, wc.TotalPower)
	valAccum := vi.GetValAccum(wc.Height, wc.ValPower)

	if valAccum.GT(totalValAccum) {
		panic("individual accum should never be greater than the total")
	}
	withdrawalTokens := fp.ValPool.MulDec(valAccum).QuoDec(totalValAccum)
	commission := withdrawalTokens.MulDec(wc.CommissionRate)
	commissionPool := vi.ValCommission.Plus(commission)
	return commissionPool
}

// get the validator's commission pool rewards at this current state,
func (vi ValidatorDistInfo) String() string {

	resp := "{\n"
	resp += fmt.Sprintf("\tOperator Address: %s\n", vi.OperatorAddr)
	resp += fmt.Sprintf("\tFeePoolWithdrawalHeight: %d\n", vi.FeePoolWithdrawalHeight)
	resp += fmt.Sprintf("\tDelAccum: {\n")
	resp += fmt.Sprintf("\t\tUpdateHeight: %d\n", vi.DelAccum.UpdateHeight)
	resp += fmt.Sprintf("\t\tAccum: %s\n", vi.DelAccum.Accum.String())
	resp += fmt.Sprintf("\t}\n")
	resp += fmt.Sprintf("\tdel_pool: %s\n", vi.DelPool.ToString())
	resp += fmt.Sprintf("\tval_commission: %s\n", vi.ValCommission.ToString())
	resp += fmt.Sprintf("}")

	return resp
}
