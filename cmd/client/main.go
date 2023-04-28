package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/andyleap/nostr/common"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/jessevdk/go-flags"
)

type config struct {
	Key *privkey
}

type privkey struct {
	*secp256k1.PrivateKey
}

func (p *privkey) UnmarshalJSON(buf []byte) error {
	var b []byte
	json.Unmarshal(buf, &b)
	if len(b) != 32 {
		return errors.New("invalid private key")
	}
	p.PrivateKey = secp256k1.PrivKeyFromBytes(b)
	return nil
}

func (p *privkey) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.PrivateKey.Serialize())
}

var CLI struct {
	ConfigFile string `long:"config" description:"Config file" default:"config.json"`
	Key        struct {
		Generate `command:"generate" description:"Generate a new key"`
		Show     `command:"show" description:"Show the current public key"`
	} `command:"key" description:"Manage keys"`
}

type Generate struct {
	Save bool `long:"save" description:"Save the key to the config file"`
}

func (g *Generate) Execute(args []string) error {
	key, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return err
	}
	pubHex := common.PubKeyHex(key.PubKey())
	priv := privkey{key}
	fmt.Printf("Public Key: %s\n", pubHex)
	buf, _ := priv.MarshalJSON()
	fmt.Printf("Private Key: %s\n", buf)

	if g.Save {
		cfg := loadJSONConfig(CLI.ConfigFile)
		cfg.Key = &priv
		saveJSONConfig(CLI.ConfigFile, cfg)
	}

	return nil
}

type Show struct{}

func (s *Show) Execute(args []string) error {
	cfg := loadJSONConfig(CLI.ConfigFile)
	if cfg.Key == nil {
		return errors.New("no key in config")
	}
	pubHex := common.PubKeyHex(cfg.Key.PubKey())
	fmt.Printf("Public Key: %s\n", pubHex)
	return nil
}

func main() {
	flags.Parse(&CLI)

}

func loadJSONConfig(filename string) config {
	var cfg config
	buf, err := os.ReadFile(filename)
	if err != nil {
		return cfg
	}
	json.Unmarshal(buf, &cfg)
	return cfg
}

func saveJSONConfig(filename string, cfg config) error {
	buf, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, buf, 0600)
}
