package cli

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/NPC-Chain/npcchub/app/v2/htlc"
	"github.com/NPC-Chain/npcchub/client/asset/cli"
	"github.com/NPC-Chain/npcchub/client/context"
	"github.com/NPC-Chain/npcchub/client/utils"
	"github.com/NPC-Chain/npcchub/codec"
	sdk "github.com/NPC-Chain/npcchub/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GetCmdCreateHTLC implements the create HTLC command
func GetCmdCreateHTLC(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an HTLC",
		Example: "iriscli htlc create --chain-id=<chain-id> --from=<key-name> --fee=0.3iris --to=<to> --receiver-on-other-chain=<receiver-on-other-chain> " +
			"--amount=<amount> --secret=<secret> --hash-lock=<hash-lock> --timestamp=<timestamp> --time-lock=<time-lock>",
		PreRunE: preCheckCmd,
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithLogger(os.Stdout).
				WithAccountDecoder(utils.GetAccountDecoder(cdc))
			txCtx := utils.NewTxContextFromCLI().WithCodec(cdc).
				WithCliCtx(cliCtx)

			sender, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			toAddrStr := viper.GetString(cli.FlagTo)
			toAddr, err := sdk.AccAddressFromBech32(toAddrStr)
			if err != nil {
				return err
			}

			receiverOnOtherChain := viper.GetString(FlagReceiverOnOtherChain)

			amountStr := viper.GetString(FlagAmount)
			amount, err := cliCtx.ParseCoins(amountStr)
			if err != nil {
				return err
			}

			timestamp := viper.GetInt64(FlagTimestamp)
			timeLock := viper.GetInt64(FlagTimeLock)

			secret := make([]byte, 32)
			var hashLock []byte

			flags := cmd.Flags()
			if flags.Changed(FlagHashLock) {
				hashLockStr := strings.TrimSpace(viper.GetString(FlagHashLock))
				hashLock, err = hex.DecodeString(hashLockStr)
				if err != nil {
					return err
				}
			} else {
				secretStr := strings.TrimSpace(viper.GetString(FlagSecret))
				if len(secretStr) > 0 {
					if len(secretStr) != 2*htlc.SecretLength {
						return fmt.Errorf("the secret must be %d bytes long", htlc.SecretLength)
					}

					secret, err = hex.DecodeString(secretStr)
					if err != nil {
						return err
					}
				} else {
					_, err := rand.Read(secret)
					if err != nil {
						return err
					}
				}

				hashLock = htlc.GetHashLock(secret, uint64(timestamp))
			}

			msg := htlc.NewMsgCreateHTLC(
				sender, toAddr, receiverOnOtherChain, amount,
				hashLock, uint64(timestamp), uint64(timeLock))

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			err = utils.SendOrPrintTx(txCtx, cliCtx, []sdk.Msg{msg})
			if err == nil && !flags.Changed(FlagHashLock) {
				fmt.Println("**Important** save this secret, hashLock in a safe place.")
				fmt.Println("It is the only way to claim or refund the locked coins from an HTLC")
				fmt.Println()
				fmt.Printf("Secret:      %s\nHashLock:    %s\n",
					hex.EncodeToString(secret), hex.EncodeToString(hashLock))
			}
			return err
		},
	}

	cmd.Flags().AddFlagSet(FsCreateHTLC)
	_ = cmd.MarkFlagRequired(cli.FlagTo)
	_ = cmd.MarkFlagRequired(FlagAmount)
	_ = cmd.MarkFlagRequired(FlagTimeLock)

	return cmd
}

// GetCmdClaimHTLC implements the claim HTLC command
func GetCmdClaimHTLC(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "claim",
		Short:   "Claim an opened HTLC",
		Example: "iriscli htlc claim --chain-id=<chain-id> --from=<key-name> --fee=0.3iris --hash-lock=<hash-lock> --secret=<secret>",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithLogger(os.Stdout).
				WithAccountDecoder(utils.GetAccountDecoder(cdc))
			txCtx := utils.NewTxContextFromCLI().WithCodec(cdc).
				WithCliCtx(cliCtx)

			sender, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			hashLockStr := viper.GetString(FlagHashLock)
			hashLock, err := hex.DecodeString(hashLockStr)
			if err != nil {
				return err
			}

			secretStr := viper.GetString(FlagSecret)
			secret, err := hex.DecodeString(secretStr)
			if err != nil {
				return err
			}

			msg := htlc.NewMsgClaimHTLC(sender, hashLock, secret)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return utils.SendOrPrintTx(txCtx, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().AddFlagSet(FsClaimHTLC)
	_ = cmd.MarkFlagRequired(FlagHashLock)
	_ = cmd.MarkFlagRequired(FlagSecret)

	return cmd
}

// GetCmdRefundHTLC implements the refund HTLC command
func GetCmdRefundHTLC(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "refund",
		Short:   "Refund from an expired HTLC",
		Example: "iriscli htlc refund --chain-id=<chain-id> --from=<key-name> --fee=0.3iris --hash-lock=<hash-lock>",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithLogger(os.Stdout).
				WithAccountDecoder(utils.GetAccountDecoder(cdc))
			txCtx := utils.NewTxContextFromCLI().WithCodec(cdc).
				WithCliCtx(cliCtx)

			sender, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			hashLockStr := viper.GetString(FlagHashLock)
			hashLock, err := hex.DecodeString(hashLockStr)
			if err != nil {
				return err
			}

			msg := htlc.NewMsgRefundHTLC(
				sender, hashLock)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return utils.SendOrPrintTx(txCtx, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().AddFlagSet(FsRefundHTLC)
	_ = cmd.MarkFlagRequired(FlagHashLock)

	return cmd
}

func preCheckCmd(cmd *cobra.Command, _ []string) error {
	// make sure either the secret or hash lock is provided
	flags := cmd.Flags()
	if flags.Changed(FlagSecret) && flags.Changed(FlagHashLock) {
		return fmt.Errorf("only one flag is allowed among the secret and hash lock")
	}

	return nil
}
