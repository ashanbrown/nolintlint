package nolintlint

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoLintLint(t *testing.T) {
	t.Run("when no explanation is provided", func(t *testing.T) {
		linter, _ := NewLinter(OptionNeeds(NeedsExplanation))
		expectIssues(t, linter, `
package bar

func foo() {
  bad() //nolint
  bad() //nolint //
  bad() //nolint // 
  good() //nolint // this is ok
	other() //nolintother
}`,
			"directive `//nolint` should provide explanation such as `//nolint // this is why` at testing.go:5:9",
			"directive `//nolint //` should provide explanation such as `//nolint // this is why` at testing.go:6:9",
			"directive `//nolint // ` should provide explanation such as `//nolint // this is why` at testing.go:7:9",
		)
	})

	t.Run("when no explanation is needed for a specific linter", func(t *testing.T) {
		linter, _ := NewLinter(OptionNeeds(NeedsExplanation), OptionExcludes([]string{"lll"}))
		expectIssues(t, linter, `
package bar

func foo() {
	thisIsAReallyLongLine() //nolint:lll
}`)
	})

	t.Run("when no specific linter is mentioned", func(t *testing.T) {
		linter, _ := NewLinter(OptionNeeds(NeedsSpecific))
		expectIssues(t, linter, `
package bar

func foo() {
  good() //nolint:my-linter
  bad() //nolint
}`, "directive `//nolint` should mention specific linter such as `//nolint:my-linter` at testing.go:6:9")
	})

	t.Run("when machine-readable style isn't used", func(t *testing.T) {
		linter, _ := NewLinter(OptionNeeds(NeedsMachine))
		expectIssues(t, linter, `
package bar

func foo() {
  bad() // nolint
  good() //nolint
}`, "directive `// nolint` should be written without leading space as `//nolint` at testing.go:5:9")
	})

	t.Run("extra spaces in front of directive are reported", func(t *testing.T) {
		linter, _ := NewLinter()
		expectIssues(t, linter, `
package bar

func foo() {
  bad() //  nolint
  good() // nolint
}`, "directive `//  nolint` should not have more than one leading space at testing.go:5:9")
	})

	t.Run("badly formatted linter list", func(t *testing.T) {
		linter, _ := NewLinter()
		expectIssues(t, linter, `
package bar

func foo() {
  good() // nolint:linter1,linter2
  bad()  // nolint:linter1 linter2
  bad()  // nolint: linter1,linter2
}`,
			"directive `// nolint:linter1 linter2` should match `// nolint[:<comma-separated-linters>] [// <explanation>]` at testing.go:6:10",  //nolint:lll // this is a string
			"directive `// nolint: linter1,linter2` should match `// nolint[:<comma-separated-linters>] [// <explanation>]` at testing.go:7:10", //nolint:lll // this is a string
		)
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
