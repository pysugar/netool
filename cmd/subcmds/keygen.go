package subcmds

import (
	"github.com/pysugar/netool/cmd/base"
	"github.com/pysugar/netool/cmd/internal/cli"
	"github.com/pysugar/netool/crypto/keygen"
	"github.com/spf13/cobra"
)

var keygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "Generate asymmetric key pairs",
	Long: `
Generate asymmetric key pairs for protocols that consume Curve25519:

  netool keygen x25519        # VLESS / REALITY flavour (base64.RawURLEncoding)
  netool keygen x25519 -e     # base64.StdEncoding
  netool keygen wg            # WireGuard (base64.StdEncoding)
`,
}

var x25519Cmd = &cobra.Command{
	Use:   `x25519 [-i "private key (base64.RawURLEncoding)"]`,
	Short: "Generate an x25519 key pair",
	Long: `
Generate an x25519 key pair.

Random:           netool keygen x25519
From private key: netool keygen x25519 -i "private key (base64.RawURLEncoding)"
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		std, _ := cmd.Flags().GetBool("std-encoding")
		input, _ := cmd.Flags().GetString("input")
		enc := keygen.RawURL
		if std {
			enc = keygen.Std
		}
		return emitKeyPair(cmd, enc, input)
	},
}

var wgCmd = &cobra.Command{
	Use:   `wg [-i "private key (base64.StdEncoding)"]`,
	Short: "Generate a WireGuard key pair",
	Long: `
Generate a WireGuard key pair (base64.StdEncoding).

Random:           netool keygen wg
From private key: netool keygen wg -i "private key (base64.StdEncoding)"
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		input, _ := cmd.Flags().GetString("input")
		return emitKeyPair(cmd, keygen.Std, input)
	},
}

func emitKeyPair(cmd *cobra.Command, enc keygen.Encoding, input string) error {
	kp, err := keygen.GenerateCurve25519(enc, input)
	if err != nil {
		return err
	}
	priv, pub := kp.Encode(enc)
	out := cli.NewOutput(cmd)
	if out.Format() == cli.FormatJSON {
		return out.JSON(map[string]string{"private": priv, "public": pub})
	}
	out.Text("Private key: %s\nPublic key: %s\n", priv, pub)
	return nil
}

func init() {
	x25519Cmd.Flags().BoolP("std-encoding", "e", false, "use base64.StdEncoding instead of RawURLEncoding")
	x25519Cmd.Flags().StringP("input", "i", "", "seed from existing base64-encoded private key")

	wgCmd.Flags().StringP("input", "i", "", "seed from existing base64-encoded private key")

	cli.Register(keygenCmd, x25519Cmd, wgCmd)
	base.AddSubCommands(keygenCmd)
}
