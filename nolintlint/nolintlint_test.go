package nolintlint

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoLintLint(t *testing.T) {
	t.Run("when no explanation is provided", func(t *testing.T) {
		linter := NewLinter([]string{"nolint"}, NeedsExplanation)
		expectIssues(t, linter, `
package bar

func foo() {
  bad() //nolint
  good() //nolint // this is ok
	other() //nolintother
}`, `must provide explanation for directive such as "//nolint // this is why" instead of "//nolint" at testing.go:5:9`)
	})

	t.Run("when no specific linter is mentioned", func(t *testing.T) {
		linter := NewLinter([]string{"nolint"}, NeedsSpecific)
		expectIssues(t, linter, `
package bar

func foo() {
  bad() //nolint:my-linter
  good() //nolint
}`, `must mention specific linter such as "//nolint:my-linter" instead of "//nolint" at testing.go:6:10`)
	})

	t.Run("when machine-readable style isn't used", func(t *testing.T) {
		linter := NewLinter([]string{"nolint"}, NeedsMachine)
		expectIssues(t, linter, `
package bar

func foo() {
  bad() // nolint
  good() //nolint
}`, `must use machine-style directive "//nolint" instead of "// nolint" at testing.go:5:9`)
	})

	t.Run("extra spaces in front of directive are reported", func(t *testing.T) {
		linter := NewLinter([]string{"nolint"}, 0)
		expectIssues(t, linter, `
package bar

func foo() {
  bad() //  nolint
  good() // nolint
}`, `lint directive "//  nolint" may not have more than one leading space at testing.go:5:9`)
	})
}

func expectIssues(t *testing.T, linter *Linter, contents string, issues ...string) {
	actualIssues := parseFile(t, linter, contents)
	actualIssueStrs := make([]string, 0, len(actualIssues))
	for _, i := range actualIssues {
		actualIssueStrs = append(actualIssueStrs, i.String())
	}
	assert.ElementsMatch(t, issues, actualIssueStrs)
}

func parseFile(t *testing.T, linter *Linter, contents string) []Issue {
	fset := token.NewFileSet()
	expr, err := parser.ParseFile(fset, "testing.go", contents, parser.ParseComments)
	if err != nil {
		t.Fatalf("unable to parse file contents: %s", err)
	}
	issues, err := linter.Run(fset, expr)
	if err != nil {
		t.Fatalf("unable to parse file: %s", err)
	}
	return issues
}
