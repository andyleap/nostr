package proto

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

type Event struct {
	ID        string     `json:"id"`
	PubKey    string     `json:"pubkey"`
	CreatedAt int64      `json:"created_at"`
	Kind      int64      `json:"kind"`
	Tags      [][]string `json:"tags"`
	Content   string     `json:"content"`
	Sig       string     `json:"sig"`
}

func (e *Event) CalcID() string {
	sigobj := []interface{}{
		0,
		e.PubKey,
		e.ID,
		e.CreatedAt,
		e.Kind,
		e.Tags,
		e.Content,
	}
	buf, _ := json.Marshal(sigobj)
	hash := sha256.Sum256(buf)
	return hex.EncodeToString(hash[:])
}

func must[t any](v t, err error) t {
	if err != nil {
		panic(err)
	}
	return v
}

func (e *Event) CheckSig() bool {
	id := e.CalcID()
	if id != e.ID {
		return false
	}
	k, err := hex.DecodeString(e.PubKey)
	if err != nil {
		return false
	}
	key, err := schnorr.ParsePubKey(k)
	if err != nil {
		return false
	}
	s, err := hex.DecodeString(e.Sig)
	if err != nil {
		return false
	}
	sig, err := schnorr.ParseSignature(s)
	if err != nil {
		return false
	}
	h, err := hex.DecodeString(id)
	if err != nil {
		return false
	}
	return sig.Verify(h, key)
}

func (e *Event) Sign(key *secp256k1.PrivateKey) error {
	id := e.CalcID()
	e.ID = id
	h, err := hex.DecodeString(id)
	if err != nil {
		return err
	}
	sig, err := schnorr.Sign(key, h)
	if err != nil {
		return err
	}
	e.Sig = hex.EncodeToString(sig.Serialize())
	return nil
}
