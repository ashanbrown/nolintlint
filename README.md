# nolintlint

makezero is a Go static analysis tool to find slice declarations that are not initialized with zero length and are later
used with append.

## Installation

    go get -u github.com/ashanbrown/makezero

## Usage

Similar to other Go static analysis tools (such as golint, go vet), makezero can be invoked with one or more filenames, directories, or packages named by its import path. makezero also supports the `...` wildcard.

    nolintlint packages...

### Flags
- **-set_exit_status** (default false) - Set exit status to 1 if any issues are found.
- **-machine** (default false) - Always require `//nolint` instead of `// nolint`
- **-specific** (default true) - Always require `//nolint:mylinter` instead of just `//nolint`
- **-explain** (default true) - Always require `//nolint // my explanation` instead of just `//nolint`

## Purpose

To ensure that lint exceptions has explanations.
Consider the case below:

```Go
import md5 //nolint

func run() {
  md5.New()
}
```

## TODO

Proposed that this should be part of golangci-lint itself.

## Contributing

Pull requests welcome!
