package preflightfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFields(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []string
	}{
		{"empty", "", nil},
		{"only spaces", "   \t ", nil},
		{"plain words", "env HOME", []string{"env", "HOME"}},
		{"collapses runs of whitespace", "env \t  HOME", []string{"env", "HOME"}},

		// The reason this exists: an argument containing a space could not be
		// written at all, because every space split it into another argument.
		{"double quoted argument", `env GREETING --exact "hello world"`, []string{"env", "GREETING", "--exact", "hello world"}},
		{"single quoted argument", `env GREETING --exact 'hello world'`, []string{"env", "GREETING", "--exact", "hello world"}},
		{"quotes join to adjacent text", `env X --exact pre"fix suffix"`, []string{"env", "X", "--exact", "prefix suffix"}},
		{"empty quoted argument stays an argument", `env X --exact ""`, []string{"env", "X", "--exact", ""}},

		// Regexes and JSON are ordinary arguments here, so quoting has to leave
		// their contents alone.
		{"single quotes keep backslashes literal", `cmd myapp --match '^v2\.'`, []string{"cmd", "myapp", "--match", `^v2\.`}},
		{"single quotes keep double quotes literal", `json f --exact '{"a": 1}'`, []string{"json", "f", "--exact", `{"a": 1}`}},
		{"double quotes keep single quotes literal", `env X --exact "it's"`, []string{"env", "X", "--exact", "it's"}},
		{"escaped double quote inside double quotes", `env X --exact "say \"hi\""`, []string{"env", "X", "--exact", `say "hi"`}},
		{"escaped backslash inside double quotes", `env X --exact "a\\b"`, []string{"env", "X", "--exact", `a\b`}},
		{"unknown escape is left alone inside double quotes", `env X --match "^v2\."`, []string{"env", "X", "--match", `^v2\.`}},
		{"backslash escapes a space outside quotes", `env X --exact hello\ world`, []string{"env", "X", "--exact", "hello world"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Fields(tt.line)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFields_UnterminatedQuote(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"unterminated double quote", `env X --exact "hello`},
		{"unterminated single quote", `env X --exact 'hello`},
		{"trailing backslash", `env X --exact hello\`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Fields(tt.line)
			require.Error(t, err, "an unclosed quote must be reported, not silently accepted")
		})
	}
}
