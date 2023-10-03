package cmd

import (
	"github.com/cortze/looking-up-ipfs/benchmark"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	concurrentLookups int    = 100
	lookupJobs        int    = 1
	outputPath        string = "./"
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
			Name:        "jobs",
			DefaultText: "number of jobs we want to execute in the benchmarks",
			Required:    true,
			Destination: &lookupJobs,
		},
		&cli.StringFlag{
			Name:        "output",
			DefaultText: "path to the folder where the results will be written",
			Required:    false,
			Destination: &outputPath,
		},
	},
}

func ConcurrentLookupBenchmark(ctx *cli.Context) error {
	logrus.WithFields(logrus.Fields{
		"concurrent-lookups": concurrentLookups,
		"jobs":               lookupJobs,
		"output-path":        outputPath,
	}).Info("launching Lookup Benchmark")

	benchmark, err := benchmark.NewLookupService(ctx.Context, lookupJobs, concurrentLookups, outputPath)
	if err != nil {
		return errors.Wrap(err, "creating the benchmark")
	}
	err = benchmark.Run()
	if err != nil {
		return errors.Wrap(err, "running the benchmark")
	}
	benchmark.Close()
	return nil
}
