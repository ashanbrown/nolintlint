package main

import (
	"flag"
	"go/ast"
	"log"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/ashanbrown/nolintlint/v2/nolintlint"
)

func main() {
	log.SetFlags(0) // remove log timestamp

	setExitStatus := flag.Bool("set_exit_status", false, "Set exit status to 1 if any issues are found")
	explain := flag.Bool("explain", true, "Require explanation for nolint directives")
	specific := flag.Bool("specific", true, "Require specific linters for nolint directives")
	machine := flag.Bool("machine", false, "Require machine-readable directives")
	exclude := flag.String("exclude", "", "Exclude the comma-separated linters from requiring explanations")
	directive := flag.String("directive", "nolint", "comma-separated list of nolint directives")

	flag.Parse()

	cfg := packages.Config{
		Mode: packages.NeedSyntax |
			packages.NeedName |
			packages.NeedTypes,
	}
	pkgs, err := packages.Load(&cfg, flag.Args()...)
	if err != nil {
		log.Fatalf("Could not load packages: %s", err)
	}
	var needs nolintlint.Needs
	if *explain {
		needs |= nolintlint.NeedsExplanation
	}
	if *specific {
		needs |= nolintlint.NeedsSpecific
	}
	if *machine {
		needs |= nolintlint.NeedsMachine
	}
	linter, err := nolintlint.NewLinter(
		nolintlint.OptionNeeds(needs),
		nolintlint.OptionDirectives(strings.Split(*directive, ",")),
		nolintlint.OptionExcludes(strings.Split(*exclude, ",")),
	)
	if err != nil {
		log.Fatalf("failed: %s", err)
	}

	var issues []nolintlint.Issue //nolint:prealloc // don't know how many there will be
	for _, p := range pkgs {
		nodes := make([]ast.Node, 0, len(p.Syntax))
		for _, n := range p.Syntax {
			nodes = append(nodes, n)
		}
		newIssues, err := linter.Run(p.Fset, nodes...)
		if err != nil {
			log.Fatalf("failed: %s", err)
		}
		if err != nil {
			log.Fatalf("failed: %s", err)
		}
		issues = append(issues, newIssues...)
	}

	for _, issue := range issues {
		log.Println(issue)
	}

	if *setExitStatus && len(issues) > 0 {
		os.Exit(1)
	}
}
