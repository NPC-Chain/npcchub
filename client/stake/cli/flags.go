package cli

import (
	flag "github.com/spf13/pflag"

	"github.com/NPC-Chain/npcchub/app/v1/stake/types"
)

// nolint
const (
	FlagAddressDelegator    = "address-delegator"
	FlagAddressValidator    = "address-validator"
	FlagAddressValidatorSrc = "address-validator-source"
	FlagAddressValidatorDst = "address-validator-dest"
	FlagPubKey              = "pubkey"
	FlagAmount              = "amount"
	FlagSharesAmount        = "shares-amount"
	FlagSharesPercent       = "shares-percent"

	FlagMoniker  = "moniker"
	FlagIdentity = "identity"
	FlagWebsite  = "website"
	FlagDetails  = "details"

	FlagCommissionRate = "commission-rate"

	FlagGenesisFormat = "genesis-format"
	FlagNodeID        = "node-id"
	FlagIP            = "ip"

	FlagOutputDocument = "output-document" // inspired by wget -O
)

// common flagsets to add to various functions
var (
	FsPk                = flag.NewFlagSet("", flag.ContinueOnError)
	FsAmount            = flag.NewFlagSet("", flag.ContinueOnError)
	fsShares            = flag.NewFlagSet("", flag.ContinueOnError)
	fsDescriptionCreate = flag.NewFlagSet("", flag.ContinueOnError)
	FsCommissionCreate  = flag.NewFlagSet("", flag.ContinueOnError)
	fsCommissionUpdate  = flag.NewFlagSet("", flag.ContinueOnError)
	fsDescriptionEdit   = flag.NewFlagSet("", flag.ContinueOnError)
	fsValidator         = flag.NewFlagSet("", flag.ContinueOnError)
	fsDelegator         = flag.NewFlagSet("", flag.ContinueOnError)
	fsRedelegation      = flag.NewFlagSet("", flag.ContinueOnError)
)

func init() {
	FsPk.String(FlagPubKey, "", "Go-Amino encoded hex PubKey of the validator. For Ed25519 the go-amino prepend hex is 1624de6220")
	FsAmount.String(FlagAmount, "", "Amount of coins to bond")
	fsShares.String(FlagSharesAmount, "", "Amount of source-shares to either unbond or redelegate as a positive integer or decimal")
	fsShares.String(FlagSharesPercent, "", "Percent of source-shares to either unbond or redelegate as a positive integer or decimal >0 and <=1")
	fsDescriptionCreate.String(FlagMoniker, "", "validator name")
	fsDescriptionCreate.String(FlagIdentity, "", "optional identity signature (ex. UPort or Keybase)")
	fsDescriptionCreate.String(FlagWebsite, "", "optional website")
	fsDescriptionCreate.String(FlagDetails, "", "optional details")
	fsCommissionUpdate.String(FlagCommissionRate, "", "The new commission rate percentage")
	FsCommissionCreate.String(FlagCommissionRate, "", "The initial commission rate percentage")
	fsDescriptionEdit.String(FlagMoniker, types.DoNotModifyDesc, "validator name")
	fsDescriptionEdit.String(FlagIdentity, types.DoNotModifyDesc, "optional identity signature (ex. UPort or Keybase)")
	fsDescriptionEdit.String(FlagWebsite, types.DoNotModifyDesc, "optional website")
	fsDescriptionEdit.String(FlagDetails, types.DoNotModifyDesc, "optional details")
	fsValidator.String(FlagAddressValidator, "", "bech address of the validator")
	fsDelegator.String(FlagAddressDelegator, "", "bech address of the delegator")
	fsRedelegation.String(FlagAddressValidatorSrc, "", "bech address of the source validator")
	fsRedelegation.String(FlagAddressValidatorDst, "", "bech address of the destination validator")
}
