package fix

import (
	"bytes"
	"regexp"
)

var postgresFixRegexp = regexp.MustCompile(`([^\\](\\\\)*)(\\u0000)+`)

func EscapeNullCharForPostgres(b []byte) []byte {
	return postgresFixRegexp.ReplaceAllFunc(b, func(b []byte) []byte {
		return bytes.ReplaceAll(b, []byte(`\u0000`), []byte(`<0x00>`))
	})
}
