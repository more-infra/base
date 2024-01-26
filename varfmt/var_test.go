package varfmt

import (
	"fmt"
	"testing"
)

func TestLightFormat(t *testing.T) {
	pattern := "${day} is another day, say: ${word} to the world, $ending_words\n\n\nbye!$end"
	expected := "tomorrow is another day, say: ${word} to the world, ${ending_words}\n\n\nbye!${end}"
	f := NewVarFormatter("$", func(name string) (string, error) {
		if name == "day" {
			return "tomorrow", nil
		}
		return fmt.Sprintf("${%s}", name), nil
	})
	val, err := f.Format(pattern)
	if err != nil {
		t.Fatal(err)
	}
	if val != expected {
		t.Fatal(val)
	}
}

func TestSpecialString(t *testing.T) {
	pattern := "${var1}$\n${var2}"
	expected := "1return2"
	f := NewVarFormatter("$", func(name string) (string, error) {
		switch name {
		case "var1":
			return "1", nil
		case "var2":
			return "2", nil
		case "\n":
			return "return", nil
		}
		return "", nil
	})
	val, err := f.Format(pattern)
	if err != nil {
		t.Fatal(err)
	}
	if val != expected {
		t.Fatal(val)
	}
}

func TestScopeRequired(t *testing.T) {
	pattern := "${var1}$common${var2}"
	optional := "1x2"
	required := "1$common2"
	providerFunc := func(name string) (string, error) {
		switch name {
		case "var1":
			return "1", nil
		case "var2":
			return "2", nil
		case "common":
			return "x", nil
		}
		return "", nil
	}
	f := NewVarFormatter("$", providerFunc,
		WithVarScopeSyntaxRequire(ScopeSyntaxRequired))
	val, err := f.Format(pattern)
	if err != nil {
		t.Fatal(err)
	}
	if val != required {
		t.Fatal(val)
	}

	f = NewVarFormatter("$", providerFunc,
		WithVarScopeSyntaxRequire(ScopeSyntaxOptional))
	val, err = f.Format(pattern)
	if err != nil {
		t.Fatal(err)
	}
	if val != optional {
		t.Fatal(val)
	}
}
