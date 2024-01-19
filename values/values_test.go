package values

import "testing"

type testCase struct {
	options          []Option
	pattern          []string
	inputAndExpected map[string]bool
}

func testWithCases(t *testing.T, c testCase) {
	m := NewMatcher(c.options...)
	for _, pattern := range c.pattern {
		if err := m.Append(pattern); err != nil {
			t.Errorf("Apppend pattern[%s] failed[%s]", pattern, err.Error())
		}
	}
	for input, expected := range c.inputAndExpected {
		r := m.Match(input)
		if r != expected {
			t.Errorf("Match with input[%s] return[%t] is not expected[%t]", input, r, expected)
		}
	}
}

func TestSensitive(t *testing.T) {
	testWithCases(t, testCase{
		options: []Option{WithMatchCaseSensitive(false)},
		pattern: []string{
			"First", "Second", "Third", "Fourth",
		},
		inputAndExpected: map[string]bool{
			"fiRSt":  true,
			"secOnd": true,
			"ThiRd":  true,
			"FourTH": true,
		},
	})
	testWithCases(t, testCase{
		options: []Option{WithMatchCaseSensitive(true)},
		pattern: []string{
			"First", "Second", "Third", "Fourth",
		},
		inputAndExpected: map[string]bool{
			"fiRSt":  false,
			"Second": true,
			"ThiRd":  false,
			"FourTH": false,
		},
	})
}

func TestRegex(t *testing.T) {
	testWithCases(t, testCase{
		options: []Option{WithMatchCaseSensitive(true), WithMatchLogic(MatchValuesOr)},
		pattern: []string{
			"/\\.jpg$/",
			"/https.+jpg/",
			"/http.+jpg/",
			"/.+fo\\.com\\/news\\/$/",
		},
		inputAndExpected: map[string]bool{
			"http://www.f.com/index.html":              false,
			"http://www.fo.com/news/":                  true,
			"http://www.fo.com/News/":                  false,
			"https://www.foo.com/news/index.html":      false,
			"https://www.fooo.com/index/html/news.jpg": true,
			"https://www.fooo.com/index/html/NEWS.JPG": false,
		},
	})

	testWithCases(t, testCase{
		options: []Option{WithMatchLogic(MatchValuesAnd)},
		pattern: []string{
			"/\\.jpg$/",
			"/^http(s)?/",
		},
		inputAndExpected: map[string]bool{
			"http://www.f.com/index.html":              false,
			"http://www.fo.com/news/":                  false,
			"http://www.fo.com/News/":                  false,
			"https://www.foo.com/news/index.html":      false,
			"https://www.fooo.com/index/html/news.jpg": true,
			"https://www.fooo.com/index/html/NEWS.JPG": true,
		},
	})
}

func TestWildcard(t *testing.T) {
	testWithCases(t, testCase{
		options: []Option{WithMatchCaseSensitive(true), WithMatchLogic(MatchValuesOr)},
		pattern: []string{
			"%*.jpg%",
			"%*foo*%",
			"%*index.html%",
		},
		inputAndExpected: map[string]bool{
			"http://www.f.com/index.html":              true,
			"http://www.fo.com/news/":                  false,
			"http://www.fo.com/News/":                  false,
			"https://www.foo.com/news/index.html":      true,
			"https://www.fooo.com/index/html/news.jpg": true,
			"https://www.Fooo.com/index/html/NEWS.JPG": false,
		},
	})

	testWithCases(t, testCase{
		options: []Option{WithMatchCaseSensitive(false), WithMatchLogic(MatchValuesAnd)},
		pattern: []string{
			"%*news*%",
			"%*fooo*%",
			"%https*%",
		},
		inputAndExpected: map[string]bool{
			"http://www.f.com/index.html":              false,
			"http://www.fo.com/news/":                  false,
			"http://www.fo.com/News/":                  false,
			"https://www.foo.com/news/index.html":      false,
			"https://www.fooo.com/index/html/news.jpg": true,
			"https://www.fooo.com/index/html/NEWS.JPG": true,
		},
	})
}
