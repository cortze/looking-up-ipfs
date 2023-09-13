package main

import (
    "os"
    "context"

    "github.com/cortze/looking-up-ipfs/cmd"

    log "github.com/sirupsen/logrus"
    cli "github.com/urfave/cli/v2"
)


func main() {
    lookingUpIPFS := &cli.App{
        Name:   "looking-up-ipfs",
        Usage:  "benchmarks the concurrency performance of IPFS' DHT",
        UsageText: "looking-up-ipfs [subcommands] [arguments]",
        Authors: []*cli.Author{
            {
                Name:  "Mikel Cortes (@cortze)",
                Email: "cortze@protonmail.com",
            },
        },
        Commands: []*cli.Command{
            cmd.RunCmd,
        },
    }
    err := lookingUpIPFS.RunContext(context.Background(), os.Args);
    if err != nil {
        log.Errorf("error: %v\n", err)
        os.Exit(1)
    }
    os.Exit(0)
}