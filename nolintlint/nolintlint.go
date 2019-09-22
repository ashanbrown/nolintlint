// nolintlint provides a linter for ensure that all //nolint directives are followed by explanations
package nolintlint

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"

	"github.com/pkg/errors"
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
	return fmt.Sprintf("directive `//%s` may not have more than one leading space", i.directiveWithOptionalLeadingSpace)
}

func (i MultipleLeadingSpaces) String() string { return toString(i) }

type NotMachine struct {
	BaseIssue
}

func (i NotMachine) Details() string {
	expected := strings.TrimSpace(i.directiveWithOptionalLeadingSpace)
	return fmt.Sprintf("use machine-style directive `//%s` instead of `//%s`", expected, i.directiveWithOptionalLeadingSpace)
}

func (i NotMachine) String() string { return toString(i) }

type NotSpecific struct {
	BaseIssue
}

func (i NotSpecific) Details() string {
	return fmt.Sprintf("mention specific linter such as `//%s:my-linter` instead of `//%s`", i.directiveWithOptionalLeadingSpace, i.directiveWithOptionalLeadingSpace)
}

func (i NotSpecific) String() string { return toString(i) }

type NoExplanation struct {
	BaseIssue
}

func (i NoExplanation) Details() string {
	return fmt.Sprintf("provide explanation for directive such as `//%s // this is why` instead of `//%s`", i.directiveWithOptionalLeadingSpace, i.directiveWithOptionalLeadingSpace)
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
	specific    *regexp.Regexp
	explanation *regexp.Regexp
}

//go:generate gobin -m -run github.com/launchdarkly/go-options config
type config struct {
	directives []string // describes lint directives to check
	excludes   []string // lists individual linters that don't require explanations
	needs      Needs    // indicates which linter checks to perform
}

type Linter struct {
	config
	patterns      []DirectivePatterns
	excludeByName map[string]bool
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
		directive, err := regexp.Compile(fmt.Sprintf(`^\s*%s(:\S+)?\b`, quoted))
		if err != nil {
			return nil, errors.Wrapf(err, `unable to create directive pattern for "%s"`, d)
		}
		specific, err := regexp.Compile(fmt.Sprintf(`^\s*%s:(\S+)`, quoted))
		if err != nil {
			return nil, errors.Wrapf(err, `unable to specific create directive pattern for "%s"`, d)
		}
		explanation, err := regexp.Compile(fmt.Sprintf(`^\s*%s(:\S+)?\s*//\s*\S*`, quoted))
		if err != nil {
			return nil, errors.Wrapf(err, `unable to specific create explanation pattern for "%s"`, d)
		}
		patterns = append(patterns, DirectivePatterns{
			directive:   directive,
			specific:    specific,
			explanation: explanation,
		})
	}
	excludeByName := make(map[string]bool)
	for _, e := range config.excludes {
		excludeByName[e] = true
	}

	return &Linter{
		config:        config,
		patterns:      patterns,
		excludeByName: excludeByName,
	}, nil
}

var leadingSpacePattern = regexp.MustCompile(`^//(\s*)`)

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
					matches := leadingSpacePattern.FindStringSubmatch(c.List[0].Text)
					if len(matches) == 0 {
						continue
					}
					leadingSpace := matches[1]
					directiveWithOptionalLeadingSpace := directive
					if len(leadingSpace) > 0 {
						directiveWithOptionalLeadingSpace = " " + directive
					}
					base := BaseIssue{
						directiveWithOptionalLeadingSpace: directiveWithOptionalLeadingSpace,
						position:                          fset.Position(c.Pos()),
					}

					// check for, report and eliminate leading spaces so we can check for other issues
					if strings.TrimSpace(directive) != directive {
						issues = append(issues, MultipleLeadingSpaces{BaseIssue: base})
						directive = strings.TrimSpace(directive)
					}

					if (l.needs&NeedsMachine) != 0 && strings.HasPrefix(directiveWithOptionalLeadingSpace, " ") {
						issues = append(issues, NotMachine{BaseIssue: base})
					}
					if (l.needs&NeedsSpecific) != 0 && !p.specific.MatchString(text) {
						issues = append(issues, NotSpecific{BaseIssue: base})
					}
					if (l.needs&NeedsExplanation) != 0 && !p.explanation.MatchString(text) {
						matches := p.specific.FindStringSubmatch(text)
						skip := false
						// check if we are excluding all of the mentioned linters
						if len(matches) > 0 {
							skip = true
							linters := strings.Split(matches[1], ",")
							for _, ll := range linters {
								if !l.excludeByName[ll] {
									skip = false
									break
								}
							}
						}
						if !skip {
							issues = append(issues, NoExplanation{BaseIssue: base})
						}
					}
				}
			}
		}
	}
	return issues, nil
}
