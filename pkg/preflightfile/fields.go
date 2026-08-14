package preflightfile

import (
	"errors"
	"strings"
)

// Fields splits a .preflight line into arguments, honouring single quotes,
// double quotes and backslash escapes the way a shell does for simple words.
//
// strings.Fields split on every space, so an argument containing one could not
// be written at all: `env GREETING --exact "hello world"` arrived as four
// arguments and cobra rejected the line with "accepts 1 arg(s), received 4"
// and a usage dump.
//
// Quoting rules are deliberately the small POSIX subset that argument values
// need. Single quotes are literal throughout, so a regex or a JSON document can
// be pasted in unchanged. Double quotes recognise \" and \\ and leave every
// other backslash alone, which keeps `"^v2\."` meaning what it looks like.
// There is no variable expansion, command substitution or globbing: a line is a
// list of arguments, not a shell command.
func Fields(line string) ([]string, error) {
	var (
		fields  []string
		current strings.Builder
		quote   byte // 0, '\'' or '"'
		started bool // a quote can produce an argument with no characters in it
	)

	for i := 0; i < len(line); i++ {
		c := line[i]

		switch {
		case quote == '\'':
			if c == '\'' {
				quote = 0
				continue
			}
			current.WriteByte(c)

		case quote == '"':
			// Only these two escapes are consumed. Anything else keeps its
			// backslash, so a regex does not need doubling.
			if c == '\\' && i+1 < len(line) && (line[i+1] == '"' || line[i+1] == '\\') {
				i++
				current.WriteByte(line[i])
				continue
			}
			if c == '"' {
				quote = 0
				continue
			}
			current.WriteByte(c)

		case c == '\'' || c == '"':
			quote = c
			started = true

		case c == '\\':
			if i+1 >= len(line) {
				return nil, errors.New("line ends with a trailing backslash")
			}
			i++
			current.WriteByte(line[i])
			started = true

		case c == ' ' || c == '\t':
			if started || current.Len() > 0 {
				fields = append(fields, current.String())
				current.Reset()
				started = false
			}

		default:
			current.WriteByte(c)
			started = true
		}
	}

	if quote != 0 {
		return nil, errors.New("unterminated " + quoteName(quote) + " quote")
	}

	if started || current.Len() > 0 {
		fields = append(fields, current.String())
	}

	return fields, nil
}

func quoteName(quote byte) string {
	if quote == '\'' {
		return "single"
	}
	return "double"
}
