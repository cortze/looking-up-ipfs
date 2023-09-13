package cmd

import (
	"github.com/urfave/cli/v2"
)

var (
	concurrentLookups int = 100
	lookupRounds      int = 1
)

var RunCmd = &cli.Command{
	Name:        "run",
	Description: "provides a set of CIDs to the IPFS DHT and measures the time of requesting them concurrently",
	Action:      ConcurrentLookupBenchmark,
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:        "lookups",
			DefaultText: "number of lookups we want to execute concurrently",
			Required:    true,
			Destination: &concurrentLookups,
		},
		&cli.IntFlag{
			Name:        "rounds",
			DefaultText: "number of lookups we want to execute concurrently",
			Required:    true,
			Destination: &lookupRounds,
		},
	},
}

func ConcurrentLookupBenchmark(ctx *cli.Context) error {

	return nil
}
