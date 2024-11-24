package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-batteries/shortner/app/config"
	"github.com/go-batteries/shortner/app/runners"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type SeedCmd struct {
	fs          *flag.FlagSet
	cmdName     string
	keyRange    string
	shortKeyLen int
	batchSize   int
	seedSize    string
}

func NewSeedCmd() *SeedCmd {
	cmd := &SeedCmd{
		fs:      flag.NewFlagSet("seed", flag.ExitOnError),
		cmdName: "seed",
	}

	return cmd
}

func (c *SeedCmd) SetArgs() {
	c.fs.StringVar(&c.keyRange, "keyrange", "all", "range of keys. like a-e. but not supported")
	if c.keyRange == "" {
		c.keyRange = "all"
	}

	c.fs.IntVar(&c.shortKeyLen, "keylen", 7, "length of short url keys")
	if c.shortKeyLen == 0 {
		c.shortKeyLen = 8
	}

	c.fs.IntVar(&c.batchSize, "batches", 1000, "batch size for bulk insert")
	c.fs.StringVar(&c.seedSize, "size", "12M", "count of keys to pre-populate per lower case letter in base58 scheme. Allowed values: K, M, B")
}

func (c *SeedCmd) Run(ctx context.Context, args []string) {
	if err := c.fs.Parse(args); err != nil {
		log.Fatal().Err(err).Msg("failed to parse seed args")
	}

	seedSize := config.MustParseSeedSize(c.seedSize)
	err := runners.SeedSqliteDB(ctx, c.shortKeyLen, c.batchSize, seedSize)
	if err != nil {
		log.Fatal().Msg("failed to seed database")
	}

	log.Info().Msg("seeding database complete")
}

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if len(os.Args) < 2 {
		log.Fatal().Msg("Expected 'seed' subcommands")
	}

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGABRT, syscall.SIGTERM)
	defer cancel()

	scmd := NewSeedCmd()
	scmd.SetArgs()

	switch os.Args[1] {
	case scmd.cmdName:
		scmd.Run(ctx, os.Args[2:])
	default:
		log.Fatal().Msgf("invalid command %s", os.Args[1])
	}

}
