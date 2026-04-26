package subcmds

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/pysugar/netool/cmd/base"
	"github.com/pysugar/netool/cmd/internal/cli"
	"github.com/spf13/cobra"
)

var readProtoCmd = &cobra.Command{
	Use:   `read-proto --data-file=hello.bin`,
	Short: "Decode a raw protobuf wire-format file",
	Long: `
Decode a raw protobuf binary file (no schema required).

  netool read-proto --data-file=hello.bin
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filename, _ := cmd.Flags().GetString("data-file")
		if filename == "" {
			return fmt.Errorf("--data-file is required")
		}
		data, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("read file %s: %w", filename, err)
		}
		return ParseProtobufTo(cli.NewOutput(cmd).Writer(), data)
	},
}

func init() {
	readProtoCmd.Flags().StringP("data-file", "f", "", "proto binary file")
	base.AddSubCommands(readProtoCmd)
}

// Wire types
const (
	Varint          = 0
	Fixed64         = 1
	LengthDelimited = 2
	StartGroup      = 3
	EndGroup        = 4
	Fixed32         = 5
)

// ParseProtobuf decodes raw protobuf wire-format bytes to os.Stdout.
// Kept for backwards compatibility with tests; new callers should prefer
// ParseProtobufTo so output is testable and routable.
func ParseProtobuf(data []byte) {
	_ = ParseProtobufTo(os.Stdout, data)
}

// ParseProtobufTo decodes raw protobuf wire-format bytes to w.
func ParseProtobufTo(w io.Writer, data []byte) error {
	reader := bytes.NewReader(data)

	for {
		key, err := readVarint(reader)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("read key: %w", err)
		}

		fieldNumber := key >> 3
		wireType := key & 0x7
		fmt.Fprintf(w, "Field Number: %d, Wire Type: %d\n", fieldNumber, wireType)

		switch wireType {
		case Varint:
			value, er := readVarint(reader)
			if er != nil {
				return fmt.Errorf("read varint value: %w", er)
			}
			fmt.Fprintf(w, "Varint Value: %d\n", value)
		case Fixed64:
			var value uint64
			if er := binary.Read(reader, binary.LittleEndian, &value); er != nil {
				return fmt.Errorf("read fixed64 value: %w", er)
			}
			fmt.Fprintf(w, "Fixed64 Value: %d\n", value)
		case LengthDelimited:
			length, er := readVarint(reader)
			if er != nil {
				return fmt.Errorf("read length: %w", er)
			}
			value := make([]byte, length)
			if _, er2 := io.ReadFull(reader, value); er2 != nil {
				return fmt.Errorf("read length-delimited value: %w", er2)
			}
			fmt.Fprintf(w, "Length-delimited %d Value: %s\n", length, value)
		case Fixed32:
			var value uint32
			if er := binary.Read(reader, binary.LittleEndian, &value); er != nil {
				return fmt.Errorf("read fixed32 value: %w", er)
			}
			fmt.Fprintf(w, "Fixed32 Value: %d\n", value)
		default:
			return fmt.Errorf("unsupported wire type: %d", wireType)
		}
	}
}

// binary.Uvarint(reader.Bytes())
func readVarint(reader *bytes.Reader) (uint64, error) {
	var value uint64
	var shift uint
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		value |= uint64(b&0x7F) << shift
		if b < 0x80 {
			break
		}
		shift += 7
	}
	return value, nil
}
