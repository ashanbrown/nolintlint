// nolintlint provides a linter for ensure that all //nolint directives are followed by explanations
package nolintlint

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"
)

type BaseIssue struct {
	directiveWithOptionalLeadingSpace string
	position                          token.Position
}

func (b BaseIssue) Position() token.Position {
	return b.position
}

type MultipleLeadingSpaces struct {
	BaseIssue
}

func (i MultipleLeadingSpaces) Details() string {
	return fmt.Sprintf(`lint directive "//%s" may not have more than one leading space`, i.directiveWithOptionalLeadingSpace)
}

func (i MultipleLeadingSpaces) String() string { return toString(i) }

type NotMachine struct {
	BaseIssue
}

func (i NotMachine) Details() string {
	expected := strings.TrimSpace(i.directiveWithOptionalLeadingSpace)
	return fmt.Sprintf(`must use machine-style directive "//%s" instead of "//%s"`, expected, i.directiveWithOptionalLeadingSpace)
}

func (i NotMachine) String() string { return toString(i) }

type NotSpecific struct {
	BaseIssue
}

func (i NotSpecific) Details() string {
	return fmt.Sprintf(`must mention specific linter such as "//%s:my-linter" instead of "//%s"`, i.directiveWithOptionalLeadingSpace, i.directiveWithOptionalLeadingSpace)
}

func (i NotSpecific) String() string { return toString(i) }

type NoExplanation struct {
	BaseIssue
}

func (i NoExplanation) Details() string {
	return fmt.Sprintf(`must provide explanation for directive such as "//%s // this is why" instead of "//%s"`, i.directiveWithOptionalLeadingSpace, i.directiveWithOptionalLeadingSpace)
}

func (i NoExplanation) String() string { return toString(i) }

func toString(i Issue) string {
	return fmt.Sprintf("%s at %s", i.Details(), i.Position())
}

type Issue interface {
	Details() string
	Position() token.Position
	String() string
}

type Needs uint

const (
	NeedsMachine Needs = 1 << iota
	NeedsSpecific
	NeedsExplanation
	NeedsAll = NeedsMachine | NeedsSpecific | NeedsExplanation
)

type DirectivePatterns struct {
	directive   *regexp.Regexp
	machine     *regexp.Regexp
	specific    *regexp.Regexp
	explanation *regexp.Regexp
}

type Linter struct {
	directives []string
	mode       Needs
	patterns   []DirectivePatterns
}

// NewLinter creates a linter that enforces that the provided directives fulfill the provided requirements
func NewLinter(directives []string, mode Needs) *Linter {
	patterns := make([]DirectivePatterns, 0, len(directives))

	for _, d := range directives {
		quoted := regexp.QuoteMeta(d)
		patterns = append(patterns, DirectivePatterns{
			directive:   regexp.MustCompile(fmt.Sprintf(`^\s*%s(:\S+)?\b`, quoted)),
			machine:     regexp.MustCompile(fmt.Sprintf(`^%s\b`, quoted)),
			specific:    regexp.MustCompile(fmt.Sprintf(`^\s*%s:\S+`, quoted)),
			explanation: regexp.MustCompile(fmt.Sprintf(`^\s*%s(:\S+)?\s*//\s*\S*`, quoted)),
		})
	}

	return &Linter{
		patterns:   patterns,
		directives: directives,
		mode:       mode,
	}
}

func (l Linter) Run(fset *token.FileSet, nodes ...ast.Node) ([]Issue, error) {
	var issues []Issue
	for _, node := range nodes {
		if file, ok := node.(*ast.File); ok {
			for _, c := range file.Comments {
				for _, p := range l.patterns {
					text := c.Text()
					directive := p.directive.FindString(text)
					if directive == "" {
						continue
					}
					// check for a space between the "//" and the directive
					leadingSpaces := int(c.List[0].End()) - int(c.List[0].Slash) - len(c.Text()) - 1 // will only be 0 or 1
					directiveWithOptionalLeadingSpace := strings.Repeat(" ", leadingSpaces) + directive
					base := BaseIssue{
						directiveWithOptionalLeadingSpace: directiveWithOptionalLeadingSpace,
						position:                          fset.Position(c.Pos()),
					}

					// check for, report and eliminate leading spaces so we can check for other issues
					if strings.TrimSpace(directive) != directive {
						issues = append(issues, MultipleLeadingSpaces{BaseIssue: base})
						directive = strings.TrimSpace(directive)
					}

					if (l.mode&NeedsMachine) != 0 && strings.HasPrefix(directiveWithOptionalLeadingSpace, " ") {
						issues = append(issues, NotMachine{BaseIssue: base})
					}
					if (l.mode&NeedsSpecific) != 0 && !p.specific.MatchString(text) {
						issues = append(issues, NotSpecific{BaseIssue: base})
					}
					if (l.mode&NeedsExplanation) != 0 && !p.explanation.MatchString(text) {
						issues = append(issues, NoExplanation{BaseIssue: base})
					}
				}
			}
		}
	}
	return issues, nil
}
