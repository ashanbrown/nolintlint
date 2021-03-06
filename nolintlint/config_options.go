package nolintlint

// Code generated by github.com/launchdarkly/go-options.  DO NOT EDIT.

type ApplyOptionFunc func(c *config) error

func (f ApplyOptionFunc) apply(c *config) error {
	return f(c)
}

func newConfig(options ...Option) (config, error) {
	var c config
	err := applyConfigOptions(&c, options...)
	return c, err
}

func applyConfigOptions(c *config, options ...Option) error {
	for _, o := range options {
		if err := o.apply(c); err != nil {
			return err
		}
	}
	return nil
}

type Option interface {
	apply(*config) error
}

// OptionDirectives describes lint directives to check
func OptionDirectives(o []string) ApplyOptionFunc {
	return func(c *config) error {
		c.directives = o
		return nil
	}
}

// OptionExcludes lists individual linters that don't require explanations
func OptionExcludes(o []string) ApplyOptionFunc {
	return func(c *config) error {
		c.excludes = o
		return nil
	}
}

// OptionNeeds indicates which linter checks to perform
func OptionNeeds(o Needs) ApplyOptionFunc {
	return func(c *config) error {
		c.needs = o
		return nil
	}
}
