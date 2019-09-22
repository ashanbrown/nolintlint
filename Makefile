SHELL=bash

test:
	diff <(sed 's|CURDIR|$(CURDIR)|' examples/expected_results.txt) <(go run . ./examples 2>&1)

init:
	GO111MODULE=off go get -u github.com/myitcv/gobin
