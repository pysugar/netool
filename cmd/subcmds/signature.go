package subcmds

import (
	"encoding/base64"
	"fmt"

	"github.com/pysugar/netool/authenticate/signature"
	"github.com/pysugar/netool/cmd/base"
	"github.com/pysugar/netool/cmd/internal/cli"
	"github.com/spf13/cobra"
)

var (
	signatureCmd = &cobra.Command{
		Use:   `signature`,
		Short: "Sign or verify a base64-encoded payload",
		Long: `
Sign or verify a base64-encoded payload.

  netool signature -k <key-b64> -i <input-b64>
  netool signature verify -k <key-b64> -i <input-b64> -s <signature-b64>
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			key, input, err := readKeyAndInput(cmd)
			if err != nil {
				return err
			}
			sign, err := signature.Sign(input, key)
			if err != nil {
				return fmt.Errorf("sign: %w", err)
			}
			out := cli.NewOutput(cmd)
			if out.Format() == cli.FormatJSON {
				return out.JSON(map[string]string{"signature": string(sign)})
			}
			out.Text("sign result: %s\n", sign)
			return nil
		},
	}

	signatureVerifyCmd = &cobra.Command{
		Use:   `verify`,
		Short: "Verify a signature against a payload",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, input, err := readKeyAndInput(cmd)
			if err != nil {
				return err
			}
			s, _ := cmd.Flags().GetString("signature")
			sign, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				return fmt.Errorf("invalid signature: %w", err)
			}
			ok := signature.VerifySignature(input, key, sign)
			out := cli.NewOutput(cmd)
			if out.Format() == cli.FormatJSON {
				return out.JSON(map[string]bool{"valid": ok})
			}
			out.Text("verify result: %v\n", ok)
			return nil
		},
	}
)

func readKeyAndInput(cmd *cobra.Command) (key, input []byte, err error) {
	k, _ := cmd.Flags().GetString("key")
	i, _ := cmd.Flags().GetString("input")
	key, err = base64.StdEncoding.DecodeString(k)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid key: %w", err)
	}
	input, err = base64.StdEncoding.DecodeString(i)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid input: %w", err)
	}
	return key, input, nil
}

func init() {
	signatureCmd.Flags().StringP("key", "k", "", "base64 secret key")
	signatureCmd.Flags().StringP("input", "i", "", "base64 input")

	signatureVerifyCmd.Flags().StringP("key", "k", "", "base64 secret key")
	signatureVerifyCmd.Flags().StringP("input", "i", "", "base64 input")
	signatureVerifyCmd.Flags().StringP("signature", "s", "", "expected base64 signature")

	cli.Register(signatureCmd, signatureVerifyCmd)
	base.AddSubCommands(signatureCmd)
}
