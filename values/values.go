package values

import (
	"github.com/becheran/wildmatch-go"
	"github.com/dlclark/regexp2"
	"strings"
)

// Matcher is used for strings match ,and it supports regex, wildcard.
// A Matcher includes several strings, regex patterns, wildcard patterns for building, and then accept one string for matching.
// The methods of Matcher are all not thread-safe.
type Matcher struct {
	stringValues   map[string]bool
	regexValues    []*regexp2.Regexp
	wildcardValues []*wildmatch.WildMatch
	patterns       map[string]bool
	sensitive      bool
	logic          MatchLogic
}

// NewMatcher accept options for match action.
// Set WithMatchCaseSensitive for if case-sensitive when do matching.
// Set WithMatchLogic for match action with "or" and "and" logic operation.
func NewMatcher(options ...Option) *Matcher {
	v := &Matcher{
		stringValues: make(map[string]bool),
		patterns:     make(map[string]bool),
		sensitive:    false,
		logic:        MatchValuesOr,
	}
	for _, opt := range options {
		opt(v)
	}
	return v
}

type Option func(*Matcher)
type MatchLogic int

const (
	MatchValuesOr  MatchLogic = 1
	MatchValuesAnd MatchLogic = 2
)

// WithMatchCaseSensitive defines if case-sensitive when do matching.
// The default value is false.
func WithMatchCaseSensitive(sensitive bool) Option {
	return func(v *Matcher) {
		v.sensitive = sensitive
	}
}

// WithMatchLogic defines match logic between patterns in Matcher.
// It's effective when regex or wildcard is not empty. The normal string matching will not use this option.
// The default value is MatchValuesOr.
func WithMatchLogic(logic MatchLogic) Option {
	return func(v *Matcher) {
		v.logic = logic
	}
}

// Append will insert a pattern into Matcher.
// The type of pattern is auto recognized in follow rules:
//
// /^regex$/    the pattern start and end with "/" means it's a regex.
//
// %wild*card%   the pattern start and end with "%" means it's a wildcard.
//
// string       otherwise, it's a string.
func (m *Matcher) Append(pattern string) error {
	if !m.sensitive {
		pattern = strings.ToLower(pattern)
	}
	if m.patterns[pattern] {
		return nil
	}
	m.patterns[pattern] = true
	if isRegex(pattern) {
		opt := regexp2.None
		if !m.sensitive {
			opt = opt | regexp2.IgnoreCase
		}
		regex, err := regexp2.Compile(escapeRegex(pattern), opt)
		if err != nil {
			return err
		}
		m.regexValues = append(m.regexValues, regex)
	} else if isWildcard(pattern) {
		m.wildcardValues = append(m.wildcardValues, wildmatch.NewWildMatch(escapeWildcard(pattern)))
	} else {
		m.stringValues[pattern] = true
	}
	return nil
}

// Match will do match input value param with pattern in Matcher with options.
func (m *Matcher) Match(value string) bool {
	if !m.sensitive {
		value = strings.ToLower(value)
	}
	if m.stringValues[value] {
		return true
	}
	matchCount := 0
	for _, regex := range m.regexValues {
		if matched, _ := regex.MatchString(value); matched {
			switch m.logic {
			case MatchValuesOr:
				return true
			case MatchValuesAnd:
				matchCount++
			}
		}
	}
	for _, wildcardMatcher := range m.wildcardValues {
		if wildcardMatcher.IsMatch(value) {
			switch m.logic {
			case MatchValuesOr:
				return true
			case MatchValuesAnd:
				matchCount++
			}
		}
	}
	if m.logic == MatchValuesAnd {
		return matchCount == len(m.regexValues)+len(m.wildcardValues)
	}
	return false
}

// Empty returns the pattern in Matcher if empty.
func (m *Matcher) Empty() bool {
	return len(m.stringValues) == 0 &&
		len(m.regexValues) == 0 &&
		len(m.wildcardValues) == 0
}

func escapeRegex(str string) string {
	return str[1 : len(str)-1]
}

func isRegex(str string) bool {
	if len(str) == 0 {
		return false
	}
	if len(str) > 2 &&
		str[0] == regexEscape &&
		str[len(str)-1] == regexEscape {
		return true
	}
	return false
}

func escapeWildcard(str string) string {
	return str[1 : len(str)-1]
}

func isWildcard(str string) bool {
	if len(str) == 0 {
		return false
	}
	if len(str) > 2 &&
		str[0] == wildcardEscape &&
		str[len(str)-1] == wildcardEscape {
		return true
	}
	return false
}

const (
	regexEscape    = '/'
	wildcardEscape = '%'
)
