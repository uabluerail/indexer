package fix

import (
	"testing"
)

func TestPostgresFix(t *testing.T) {
	type testCase struct{ input, want string }

	cases := []testCase{
		{`"a"`, `"a"`},
		{`"\u0000"`, `"<0x00>"`},
		{`"description":"\u0000"`, `"description":"<0x00>"`},
		{`"\\u0000"`, `"\\u0000"`},
		{`"\\\u0000"`, `"\\<0x00>"`},
		{`\n\n\u0000\u0000 \u0000\u0000\u0000\u0000 \u0000\u0000\u0000\u0000\u0000`,
			`\n\n<0x00><0x00> <0x00><0x00><0x00><0x00> <0x00><0x00><0x00><0x00><0x00>`},
	}

	for _, tc := range cases {
		got := EscapeNullCharForPostgres([]byte(tc.input))
		if string(got) != tc.want {
			t.Errorf("escapeNullCharForPostgres(%s) = %s, want %s", tc.input, string(got), tc.want)
		}
	}
}
