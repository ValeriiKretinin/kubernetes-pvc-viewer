package matcher

import "github.com/bmatcuk/doublestar/v4"

type Matcher struct{ include, exclude []string }

func New(include, exclude []string) Matcher { return Matcher{include: include, exclude: exclude} }

func (m Matcher) Match(s string) bool {
	// empty include => no match by default per spec
	if len(m.include) == 0 {
		return false
	}
	included := false
	for _, p := range m.include {
		if ok, _ := doublestar.Match(p, s); ok {
			included = true
			break
		}
	}
	if !included {
		return false
	}
	for _, p := range m.exclude {
		if ok, _ := doublestar.Match(p, s); ok {
			return false
		}
	}
	return true
}
