package clicommand

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/buildkite/agent/v3/internal/jwkutil"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/urfave/cli"
	"golang.org/x/exp/slices"
)

type KeygenConfig struct {
	Alg                   string `cli:"alg" validate:"required"`
	KeyID                 string `cli:"key-id" validate:"required"`
	PrivateKeySetFilename string `cli:"private-keyset-filename" normalize:"filepath"`
	PublicKeysetFilename  string `cli:"public-keyset-filename" normalize:"filepath"`

	NoColor     bool     `cli:"no-color"`
	Debug       bool     `cli:"debug"`
	LogLevel    string   `cli:"log-level"`
	Experiments []string `cli:"experiment"`
	Profile     string   `cli:"profile"`
}

var KeygenCommand = cli.Command{
	Name:  "keygen",
	Usage: "Generate a new key pair, used to sign and verify jobs",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:   "alg",
			EnvVar: "BUILDKITE_AGENT_KEYGEN_ALG",
			Usage:  fmt.Sprintf("The JWS signing algorithm to use for the key pair. Valid algorithms are: %v", ValidSigningAlgorithms),
		},
		cli.StringFlag{
			Name:   "key-id",
			EnvVar: "BUILDKITE_AGENT_KEYGEN_KEY_ID",
			Usage:  "The ID to use for the keys generated",
		},
		cli.StringFlag{
			Name:   "private-keyset-filename",
			EnvVar: "BUILDKITE_AGENT_KEYGEN_PRIVATE_KEY_FILENAME",
			Usage:  "The filename to write the private key to. Defaults to a name based on the key id in the current directory",
		},
		cli.StringFlag{
			Name:   "public-keyset-filename",
			EnvVar: "BUILDKITE_AGENT_KEYGEN_PUBLIC_KEYSET_FILENAME",
			Usage:  "The filename to write the public keyset to. Defaults to a name based on the key id in the current directory",
		},

		// Global flags
		NoColorFlag,
		DebugFlag,
		LogLevelFlag,
		ExperimentsFlag,
		ProfileFlag,
	},
	Action: func(c *cli.Context) {
		_, cfg, l, _, done := setupLoggerAndConfig[KeygenConfig](context.Background(), c)
		defer done()

		sigAlg := jwa.SignatureAlgorithm(cfg.Alg)

		if !slices.Contains(ValidSigningAlgorithms, sigAlg) {
			l.Fatal("Invalid signing algorithm: %s. Valid signing algorithms are: %s", cfg.Alg, ValidSigningAlgorithms)
		}

		priv, pub, err := jwkutil.NewKeyPair(cfg.KeyID, sigAlg)
		if err != nil {
			l.Fatal("Failed to generate key pair: %v", err)
		}

		if cfg.PrivateKeySetFilename == "" {
			cfg.PrivateKeySetFilename = fmt.Sprintf("./%s-%s-private.json", cfg.Alg, cfg.KeyID)
		}

		if cfg.PublicKeysetFilename == "" {
			cfg.PublicKeysetFilename = fmt.Sprintf("./%s-%s-public.json", cfg.Alg, cfg.KeyID)
		}

		l.Info("Writing private key set to %s...", cfg.PrivateKeySetFilename)
		pKey, err := json.Marshal(priv)
		if err != nil {
			l.Fatal("Failed to marshal private key: %v", err)
		}

		err = writeIfNotExists(cfg.PrivateKeySetFilename, pKey)
		if err != nil {
			l.Fatal("Failed to write private key file: %v", err)
		}

		l.Info("Writing public key set to %s...", cfg.PublicKeysetFilename)
		pubKey, err := json.Marshal(pub)
		if err != nil {
			l.Fatal("Failed to marshal private key: %v", err)
		}

		err = writeIfNotExists(cfg.PublicKeysetFilename, pubKey)
		if err != nil {
			l.Fatal("Failed to write private key file: %v", err)
		}

		l.Info("Done! Enjoy your new keys ^_^")

		if slices.Contains(ValidOctetAlgorithms, sigAlg) {
			l.Info("Note: Because you're using the %s algorithm, which is symmetric, the public and private keys are identical", sigAlg)
		}
	},
}

func writeIfNotExists(filename string, data []byte) error {
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("file %s already exists", filename)
	}

	return os.WriteFile(filename, data, 0o600)
}
