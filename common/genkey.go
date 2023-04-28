package common

import "github.com/decred/dcrd/dcrec/secp256k1/v4"

func GeneratePrivateKey() *secp256k1.PrivateKey {
	for {
		key, _ := secp256k1.GeneratePrivateKey()
		if key.PubKey().SerializeCompressed()[0] == 2 {
			return key
		}
	}
}
