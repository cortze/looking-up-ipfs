# looking-up-ipfs
Wondering how well does IPFS' DHT code perform doing multiple operations concurrently? Multiple provides and retreivals? This repo contains the tools that you really need!

## Features

- Basic command line cli to configure the study.
- Custom Fork of `go-libp2p-kad-dht` to extract metrics from the most common used operations.
- Basic `csv` exporter for the individual operations + the aggragation of the jobs.
- Simple visualization scripts in Jupyter Notebook.

## Installation

You can install **looking-up-ipfs** using `make`:

```bash
make dependencies
make build
```

## Usage
Here's how you can use the tool from the command line:

```shell
NAME:
   looking-up-ipfs - benchmarks the concurrency performance of IPFS DHT

USAGE:
   looking-up-ipfs [subcommands] [arguments]

COMMANDS:
   run      
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help
```

Example: 
```shell
mkdir results
./build/looking-up-ipfs run --lookups 75 --jobs 100 --output results/test
```

will produce something like this:
```
results/
├── 100_jobs_indv_provide_summary.csv
├── 100_jobs_indv_retrieval_summary.csv
├── 100_jobs_provide_jobs_summary.csv
└── 100_jobs_retrieval_jobs_summary.csv
```

## Contributing
If you want to contribute to this project, please follow these guidelines:

- Fork the repository
- Create a new branch
- Make your changes
- Submit a pull request
- Bugs and Issues

If you encounter any bugs or issues, please report them here.

## Contact
Author: Mikel Cortes ([@cortze](https://github.com/cortze))

Feel free to reach out if you have any questions or suggestions!

## License
This project is licensed under the MIT License - see the [LICENSE](https://github.com/cortze/looking-up-ipfs/blob/main/LICENSE) file for details.
