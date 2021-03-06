// nolintlint provides a linter for ensure that all //nolint directives are followed by explanations
package nolintlint

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

type BaseIssue struct {
	fullDirective                     string
	directiveWithOptionalLeadingSpace string
	position                          token.Position
}

func (b BaseIssue) Position() token.Position {
	return b.position
}

type ExtraLeadingSpace struct {
	BaseIssue
}

func (i ExtraLeadingSpace) Details() string {
	return fmt.Sprintf("directive `%s` should not have more than one leading space", i.fullDirective)
}

func (i ExtraLeadingSpace) String() string { return toString(i) }

type NotMachine struct {
	BaseIssue
}

func (i NotMachine) Details() string {
	expected := i.fullDirective[:2] + strings.TrimLeftFunc(i.fullDirective[2:], unicode.IsSpace)
	return fmt.Sprintf("directive `%s` should be written without leading space as `%s`",
		i.fullDirective, expected)
}

func (i NotMachine) String() string { return toString(i) }

type NotSpecific struct {
	BaseIssue
}

func (i NotSpecific) Details() string {
	return fmt.Sprintf("directive `%s` should mention specific linter such as `//%s:my-linter`",
		i.fullDirective, i.directiveWithOptionalLeadingSpace)
}

func (i NotSpecific) String() string { return toString(i) }

type ParseError struct {
	BaseIssue
}

func (i ParseError) Details() string {
	return fmt.Sprintf("directive `%s` should match `//%s[:<comma-separated-linters>] [// <explanation>]`",
		i.fullDirective,
		i.directiveWithOptionalLeadingSpace)
}

func (i ParseError) String() string { return toString(i) }

type NoExplanation struct {
	BaseIssue
	fullDirectiveWithoutExplanation string
}

func (i NoExplanation) Details() string {
	return fmt.Sprintf("directive `%s` should provide explanation such as `%s // this is why`",
		i.fullDirective, i.fullDirectiveWithoutExplanation)
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
	directive *regexp.Regexp
	full      *regexp.Regexp
}

//go:generate gobin -m -run github.com/launchdarkly/go-options config
type config struct {
	directives []string // describes lint directives to check
	excludes   []string // lists individual linters that don't require explanations
	needs      Needs    // indicates which linter checks to perform
}

type Linter struct {
	config
	patterns        []DirectivePatterns
	excludeByLinter map[string]bool
}

var defaultDirectives = []string{"nolint"}

// NewLinter creates a linter that enforces that the provided directives fulfill the provided requirements
func NewLinter(options ...Option) (*Linter, error) {
	config, err := newConfig(append([]Option{OptionDirectives(defaultDirectives)}, options...)...)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse options")
	}
	patterns := make([]DirectivePatterns, 0, len(config.directives))
	for _, d := range config.directives {
		quoted := regexp.QuoteMeta(d)
		directive, err := regexp.Compile(fmt.Sprintf(`^\s*(%s)(:\S+)?\b`, quoted))
		if err != nil {
			return nil, errors.Wrapf(err, `unable to create directive pattern for "%s"`, d)
		}
		full, err := regexp.Compile(fmt.Sprintf(`^//\s*%s(:\S+)?\s*(//.*)?\s*\n?$`, quoted))
		if err != nil {
			return nil, errors.Wrapf(err, `unable to specific create full pattern for "%s"`, d)
		}
		patterns = append(patterns, DirectivePatterns{
			directive: directive,
			full:      full,
		})
	}
	excludeByName := make(map[string]bool)
	for _, e := range config.excludes {
		excludeByName[e] = true
	}

	return &Linter{
		config:          config,
		patterns:        patterns,
		excludeByLinter: excludeByName,
	}, nil
}

var leadingSpacePattern = regexp.MustCompile(`^//(\s*)`)
var trailingBlankExplanation = regexp.MustCompile(`\s*(//\s*)?$`)

func (l Linter) Run(fset *token.FileSet, nodes ...ast.Node) ([]Issue, error) {
	var issues []Issue
	for _, node := range nodes {
		if file, ok := node.(*ast.File); ok {
			for _, c := range file.Comments {
				for _, p := range l.patterns {
					text := c.Text()
					matches := p.directive.FindStringSubmatch(text)
					if len(matches) == 0 {
						continue
					}
					directive := matches[1]

					// check for a space between the "//" and the directive
					leadingSpaceMatches := leadingSpacePattern.FindStringSubmatch(c.List[0].Text) // c.Text() doesn't have all leading space
					if len(leadingSpaceMatches) == 0 {
						continue
					}
					leadingSpace := leadingSpaceMatches[1]

					directiveWithOptionalLeadingSpace := directive
					if len(leadingSpace) > 0 {
						directiveWithOptionalLeadingSpace = " " + directive
					}

					base := BaseIssue{
						fullDirective:                     c.List[0].Text,
						directiveWithOptionalLeadingSpace: directiveWithOptionalLeadingSpace,
						position:                          fset.Position(c.Pos()),
					}

					// check for, report and eliminate leading spaces so we can check for other issues
					if leadingSpace != "" && leadingSpace != " " {
						issues = append(issues, ExtraLeadingSpace{
							BaseIssue: base,
						})
					}

					if (l.needs&NeedsMachine) != 0 && strings.HasPrefix(directiveWithOptionalLeadingSpace, " ") {
						issues = append(issues, NotMachine{BaseIssue: base})
					}

					fullMatches := p.full.FindStringSubmatch(c.List[0].Text)
					if len(fullMatches) == 0 {
						issues = append(issues, ParseError{BaseIssue: base})
						continue
					}
					lintersText, explanation := fullMatches[1], fullMatches[2]
					var linters []string
					if len(lintersText) > 0 {
						lls := strings.Split(lintersText[1:], ",")
						linters = make([]string, 0, len(lls))
						for _, ll := range lls {
							if ll != "" {
								linters = append(linters, ll)
								break
							}
						}
					}
					if (l.needs & NeedsSpecific) != 0 {
						if len(linters) == 0 {
							issues = append(issues, NotSpecific{BaseIssue: base})
						}
					}

					if (l.needs&NeedsExplanation) != 0 && (explanation == "" || strings.TrimSpace(explanation) == "//") {
						needsExplanation := len(linters) == 0 // if no linters are mentioned, we must have explanation
						// otherwise, check if we are excluding all of the mentioned linters
						for _, ll := range linters {
							if !l.excludeByLinter[ll] { // if a linter does require explanation
								needsExplanation = true
								break
							}
						}
						if needsExplanation {
							fullDirectiveWithoutExplanation := trailingBlankExplanation.ReplaceAllString(c.List[0].Text, "")
							issues = append(issues, NoExplanation{
								BaseIssue:                       base,
								fullDirectiveWithoutExplanation: fullDirectiveWithoutExplanation,
							})
						}
					}
				}
			}
		}
	}
	return issues, nil
}
