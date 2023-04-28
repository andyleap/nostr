package common

import (
	"encoding/hex"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

func PubKeyHex(pubKey *secp256k1.PublicKey) string {
	buf := make([]byte, 32)
	pubKey.X().FillBytes(buf)
	return hex.EncodeToString(buf)
}
