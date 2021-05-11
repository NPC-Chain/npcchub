package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/NPC-Chain/npcchub/types"
)

// test ValidateBasic for MsgEditValidator
func TestMsgWithdrawDelegatorReward(t *testing.T) {
	tests := []struct {
		delegatorAddr sdk.AccAddress
		validatorAddr sdk.ValAddress
		expectPass    bool
	}{
		{delAddr1, valAddr1, true},
		{emptyDelAddr, valAddr1, false},
		{delAddr1, emptyValAddr, false},
		{emptyDelAddr, emptyValAddr, false},
	}
	for i, tc := range tests {
		msg := NewMsgWithdrawDelegatorReward(tc.delegatorAddr, tc.validatorAddr)
		if tc.expectPass {
			require.Nil(t, msg.ValidateBasic(), "test index: %v", i)
		} else {
			require.NotNil(t, msg.ValidateBasic(), "test index: %v", i)
		}
	}
}

// test ValidateBasic and GetSigners for MsgCreateValidatorOnBehalfOf
func TestMsgWithdrawDelegatorRewardsAll(t *testing.T) {
	tests := []struct {
		delegatorAddr sdk.AccAddress
		expectPass    bool
	}{
		{delAddr1, true},
		{emptyDelAddr, false},
	}
	for i, tc := range tests {
		msg := NewMsgWithdrawDelegatorRewardsAll(tc.delegatorAddr)
		if tc.expectPass {
			require.Nil(t, msg.ValidateBasic(), "test index: %v", i)
		} else {
			require.NotNil(t, msg.ValidateBasic(), "test index: %v", i)
		}
	}
}

// test ValidateBasic for MsgDelegate
func TestMsgWithdrawValidatorRewardsAll(t *testing.T) {
	tests := []struct {
		validatorAddr sdk.ValAddress
		expectPass    bool
	}{
		{valAddr1, true},
		{emptyValAddr, false},
	}
	for i, tc := range tests {
		msg := NewMsgWithdrawValidatorRewardsAll(tc.validatorAddr)
		if tc.expectPass {
			require.Nil(t, msg.ValidateBasic(), "test index: %v", i)
		} else {
			require.NotNil(t, msg.ValidateBasic(), "test index: %v", i)
		}
	}
}
