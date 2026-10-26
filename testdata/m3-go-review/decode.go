package fixture

import (
	"fmt"
	"github.com/stellar/go-stellar-sdk/strkey"
	"strings"
)

func decodeDestination(address string) (*strkey.MuxedContract, error) {
	if !strings.HasPrefix(address, "C") {
		return nil, fmt.Errorf("unsupported address prefix")
	}
	return strkey.DecodeMuxedContract(address)
}
