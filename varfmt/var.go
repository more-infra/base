package varfmt

import (
	"errors"
	"fmt"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/more-infra/base"
	"strings"
	"sync"
)

const (
	ErrTypeVarStringInvalid = "varfmt.var_string_invalid"
)

var (
	ErrVarStringEmpty   = errors.New("var string is empty for parsing")
	ErrVarFormatInvalid = errors.New("var format is invalid for paring")
)

// Formatter used to format vars in string with the syntax by self-defines.
// Usually, the variable is format as "${var_name}" or "$var_name" and "@var(var_name)...",
// with Formatter, you can define the prefix or scope by input params.
// Formatter will parse the string and acquire variables in string, then replace the value of variable by provider input.
type Formatter struct {
	prefixSyntax       string
	scopeSyntaxRequire ScopeSyntaxRequire
	scopeSyntax        ScopeSyntax
	pvd                VarProvider
	cache              *VarParseCache
}

type VarProvider func(string) (string, error)

type Option func(*Formatter)

type ScopeSyntaxRequire string

func (t ScopeSyntaxRequire) String() string {
	return string(t)
}

const (
	// ScopeSyntaxOptional means the vars could use the $var_name or ${var_name} format
	ScopeSyntaxOptional ScopeSyntaxRequire = "optional"

	// ScopeSyntaxRequired means the vars must be ${var_name} format
	ScopeSyntaxRequired ScopeSyntaxRequire = "required"
)

type ScopeSyntax string

func (t ScopeSyntax) String() string {
	return string(t)
}

const (
	// ScopeSyntaxBrace defines the vars format ${var_name}, $ is the prefix-syntax input of NewVarFormatter
	ScopeSyntaxBrace ScopeSyntax = "brace"

	// ScopeSyntaxParentheses defines the vars format $(var_name) or @func(var_name), $ or @func is the prefix-syntax.
	ScopeSyntaxParentheses ScopeSyntax = "parentheses"
)

type ScopeSyntaxArchive struct {
	Begin byte
	End   byte
}

var (
	ScopeSyntaxValue = map[ScopeSyntax]ScopeSyntaxArchive{
		ScopeSyntaxBrace:       {'{', '}'},
		ScopeSyntaxParentheses: {'(', ')'},
	}
)

// NewVarFormatter create the formatter with required params and optional options.
//
// prefixSyntax is usually "$", or you could define it yourself, such as "#", "&", even words as "@encrypt", "@decrypt"
//
// pvd defines the value provider by variable name.
//
// options define the flexible syntax of variable.
//
// a cache is required as default, when do WithVarParseCache is not set, a default cache is given.See NewVarParseCache for more details.
func NewVarFormatter(prefixSyntax string, pvd VarProvider, options ...Option) *Formatter {
	formatter := &Formatter{
		prefixSyntax:       prefixSyntax,
		scopeSyntaxRequire: ScopeSyntaxOptional,
		scopeSyntax:        ScopeSyntaxBrace,
		pvd:                pvd,
	}
	for _, option := range options {
		option(formatter)
	}
	if formatter.cache == nil {
		formatter.cache = NewVarParseCache()
	}
	return formatter
}

// WithVarParseCache defines the self-defined cache.
// The default value is not nil but a default cache.
func WithVarParseCache(cache *VarParseCache) Option {
	return func(formatter *Formatter) {
		formatter.cache = cache
	}
}

// WithVarScopeSyntaxRequire defines if the scope is required of variable syntax.
func WithVarScopeSyntaxRequire(require ScopeSyntaxRequire) Option {
	return func(formatter *Formatter) {
		formatter.scopeSyntaxRequire = require
	}
}

// WithVarScopeSyntax defines the scope value of variable syntax.
// It supports "{}" or "()" now.
func WithVarScopeSyntax(syntax ScopeSyntax) Option {
	return func(formatter *Formatter) {
		formatter.scopeSyntax = syntax
	}
}

// Format will replace all variable conformed the syntax to the value by VarProvider.
// When the VarProvider return error with the variable, the Format will interrupt and return error.
func (fm *Formatter) Format(str string) (string, error) {
	var (
		scheme *varScheme
		err    error
	)
	if -1 == strings.Index(str, fm.prefixSyntax) {
		return str, nil
	}
	scheme = fm.cache.get(str, fm.prefixSyntax, func(s string) *varScheme {
		return fm.parse(str)
	})
	err = scheme.error()
	if err != nil {
		return "", err
	}
	return scheme.evaluate(fm.pvd)
}

func (fm *Formatter) parse(str string) *varScheme {
	va := strings.Split(str, fm.prefixSyntax)
	scheme := newScheme()
	scheme.push(&field{
		s: va[0],
	})
	for n := 1; n != len(va); n++ {
		f, err := fm.parseField(va[n])
		if err != nil {
			return newErrorScheme(err.WithField("string", str))
		}
		scheme.push(f)
	}
	return scheme
}

func (fm *Formatter) parseField(f string) (*field, *base.Error) {
	if len(f) == 0 {
		return nil, base.NewErrorWithType(ErrTypeVarStringInvalid, ErrVarStringEmpty).
			WithMessage("empty var name found")
	}
	var (
		v string
		s string
	)
	if f[0] != ScopeSyntaxValue[fm.scopeSyntax].Begin {
		if fm.scopeSyntaxRequire == ScopeSyntaxRequired {
			s = fm.prefixSyntax + f
		} else {
			va := strings.Fields(f)
			if len(va) == 0 {
				v = f
			} else {
				v = va[0]
				if len(f) > len(v) {
					s = f[len(v):]
				}
			}
		}
	} else {
		r := strings.Index(f, string(ScopeSyntaxValue[fm.scopeSyntax].End))
		if r == -1 {
			return nil, base.NewErrorWithType(ErrTypeVarStringInvalid, ErrVarFormatInvalid).
				WithMessage(fmt.Sprintf("syntax %s not found", string(ScopeSyntaxValue[fm.scopeSyntax].End))).
				WithField("field", f)
		}
		v = f[1:r]
		s = f[r+1:]
	}
	return &field{
		v: v,
		s: s,
	}, nil
}

// VarParseCache save the parsed syntax result of string by a lru cache.
// It prevents formatter for parsing the string syntax repeatedly.
type VarParseCache struct {
	mu       sync.Mutex
	capacity int
	cache    *lru.Cache[string, *varScheme]
}

// NewVarParseCache create the cache for string syntax parsed result.
// It is used in WithVarParseCache, which is NewVarFormatter options params.
//
// options provide the capacity of lru cache defined.
func NewVarParseCache(options ...CacheOption) *VarParseCache {
	c := &VarParseCache{
		capacity: 128,
	}
	for _, option := range options {
		option(c)
	}
	cache, err := lru.New[string, *varScheme](c.capacity)
	if err != nil {
		panic(err)
	}
	c.cache = cache
	return c
}

type CacheOption func(c *VarParseCache)

// WithCacheCapacity defines the capacity of the lru cache.
func WithCacheCapacity(capacity int) CacheOption {
	return func(c *VarParseCache) {
		c.capacity = capacity
	}
}

func (c *VarParseCache) get(str string, syntax string, creator func(string) *varScheme) *varScheme {
	key := syntax + "->" + str
	v, ok := c.cache.Get(key)
	if ok {
		return v
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok = c.cache.Get(key)
	if ok {
		return v
	}
	v = creator(str)
	c.cache.Add(key, v)
	return v
}

type field struct {
	v string
	s string
}

type varScheme struct {
	err    error
	fields []*field
}

func newScheme() *varScheme {
	return &varScheme{}
}

func newErrorScheme(err error) *varScheme {
	return &varScheme{
		err: err,
	}
}

func (s *varScheme) push(f *field) {
	s.fields = append(s.fields, f)
}

func (s *varScheme) error() error {
	return s.err
}

func (s *varScheme) evaluate(pvd VarProvider) (string, error) {
	var value string
	for _, f := range s.fields {
		if len(f.v) == 0 {
			value += f.s
			continue
		}
		v, err := pvd(f.v)
		if err != nil {
			return "", err
		}
		value += v + f.s
	}
	return value, nil
}
