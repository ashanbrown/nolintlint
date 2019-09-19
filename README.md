# nolintlint

nolintlint is a Go static analysis tool to find ill-formed or insufficiently explained `// nolint` directives for golangci
(or any other linter)

## Installation

    go get -u github.com/ashanbrown/nolintlint

## Usage

    nolintlint [flags...] packages...

### Flags

- **-set_exit_status** (default false) - Set exit status to 1 if any issues are found.
- **-machine** (default false) - Always require `//nolint` instead of `// nolint`
- **-specific** (default true) - Always require `//nolint:mylinter` instead of just `//nolint`
- **-explain** (default true) - Always require `//nolint // my explanation` instead of just `//nolint`

## Purpose

To ensure that lint exceptions has explanations.  Consider the case below:

```Go
import "crypto/md5" //nolint

func hash(data []byte) []byte {
	return md5.New().Sum(data) //nolint
}
```

In the above case, nolint directives are present but the user has no idea why this is being done or which linter
is being suppressed (in this case, gosec recommends against use of md5). 

`nolintlint` can also identify cases where you may have written `//  nolint`.  Finally nolintlint, can enforce that you
use the machine-readable nolint directive format `//nolint`.

## TODO

Propose that this should be part of golangci-lint itself.

## Contributing

Pull requests welcome!
