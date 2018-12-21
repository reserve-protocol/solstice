# 🌞 Solstice 🌞

Solstice is a code coverage tool for the solidity language. The current dependencies are;
* A running [parity client](https://www.parity.io/ethereum/)
* A local version of [`solc`](https://solidity.readthedocs.io/en/latest/installing-solidity.html)
* Access to the solidity contract code (and not just the addresses of said contracts on the blockchain).

It produces html files of the marked-up source code, where covered regions are marked green and uncovered regions are red.

## Rudimentary setup instructions
* Clone this repo
* Have a working [go](https://golang.org/doc/install) development environment
* Run `dep ensure` just inside the directory to install dependencies
* Get a parity blockchain running
* Fill out the `config.yml` file
* Run `go build -o solstice`
* Run `./solstice cover`

If all works as intended, your test command will run, you will see transactions happening on the parity client, and html files will be produced inside the `coverage_report_dir` you specified in the config file. If you open those html files in a browser, they should looks like your source code files, marked red or green in a reasonable pattern reflecting your test coverage.

## The config file
Solstice supports the following configuration options in a YAML file. For an example, see `config.yml`.
* `contracts_dir`: A directory which contains all of your `.sol` contract files.
* `coverage_report_dir`: The directory that you want the coverage report to end up in.
* `blockchain_client`: The URL and port that your parity blockchain client is available on.
* `test_command`: The command that runs your testing suite, which will send transactions to `blockchain_client`. Each space-separated part of the command should go on a separate line in the yaml, as a list.
* `solc_args`: A YAML list of args to be given to the solc compiler while compiling your contracts. These args will be placed between the `solc` invocation and the `--combined-json` flag, in the order given. These args should match the ones that were originally used to compile the contracts that the `test_command` sends transactions to.

## Other commands
`solstice debug` will tell you the last line of code that a particular transaction ended on. This is especially useful for reverts, since the EVM does not currently provide any kind of error messages or stack traces.

`solstice display` has two modes. One takes a transaction ID and delivers marked up source code for each step in the transaction, similar to a stack trace. The other takes a contract file and delivers marked up source code for each node in the abstract syntax tree (AST) of that file.

`solstice cover_line` prints a more simplistic report of contract line numbers that were hit during the test run.

## Running the tests

Some unit tests are available by running `go test ./tests`.
