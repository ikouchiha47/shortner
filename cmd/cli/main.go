package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-batteries/shortner/app/config"
	"github.com/go-batteries/shortner/app/db"
	"github.com/go-batteries/shortner/app/models"
	"github.com/go-batteries/shortner/app/runners"
	"github.com/go-batteries/shortner/app/seed"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ArrayFlags []string

func (i *ArrayFlags) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *ArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

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

type ProbeCmd struct {
	fs       *flag.FlagSet
	cmdName  string
	keyRange string

	coord  *db.SqliteCoordinator[string]
	seeder *seed.Seeder
}

func NewProbCmd() *ProbeCmd {
	cmd := &ProbeCmd{
		fs:      flag.NewFlagSet("probe", flag.ExitOnError),
		cmdName: "probe",
	}

	seeder := seed.RegisterUrlSeeder()

	cmd.seeder = seeder

	return cmd
}

func (c *ProbeCmd) SetArgs() {
	c.fs.StringVar(&c.keyRange, "keyrange", "a-z", "key ranges can be a-z or a-e. This takes care of all")
}

func (c *ProbeCmd) Run(ctx context.Context, args []string) {
	if err := c.fs.Parse(args); err != nil {
		log.Fatal().Err(err).Msg("failed to parse probe cmds")
	}

	start, end, ok := models.ExplodeKeyRange(c.keyRange)
	if !ok {
		log.Fatal().Str("keyRange", c.keyRange).Msg("invalid key range. expected format start-end")
	}

	keyRanges := c.seeder.Shards(5)

	// this is an interval problem
	// given keyranges, a-e, f-p, q-s
	// and provided range b-g
	// find the intervals that intersect

	filteredRanges := []string{}

	for _, keyRange := range keyRanges {
		s, e, _ := models.ExplodeKeyRange(keyRange)

		startInRange := start >= s && start <= e
		endInRange := end >= s && end <= e

		if startInRange || endInRange {
			filteredRanges = append(filteredRanges, keyRange)
		}
	}

	database := db.NewSqliteCoordinator(filteredRanges)

	err := database.ConnectShards(ctx, db.DBReadOnlyMode)
	if err != nil {
		log.Fatal().Msg("failed to connect to databases")
	}

	shards, ok := database.GetShards()
	if !ok {
		log.Fatal().Msg("should not have failed to create shards")
	}

	for _, shard := range shards {
		repo := models.NewProber(shard.ShardKey(), shard.Conn(), models.URLKeysProberQuery)
		stats, err := repo.GetStats(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to get db status")
		} else {
			fmt.Printf("shard: %s, stats: %v", shard.ShardKey(), *stats)
		}
	}
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

	pcmd := NewProbCmd()
	pcmd.SetArgs()

	switch os.Args[1] {
	case scmd.cmdName:
		scmd.Run(ctx, os.Args[2:])
	case pcmd.cmdName:
		pcmd.Run(ctx, os.Args[2:])
	default:
		log.Fatal().Msgf("invalid command %s", os.Args[1])
	}

}
