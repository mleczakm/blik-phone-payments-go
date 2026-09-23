package payments

import (
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"
)

const (
	// CodeLength is the length of payment codes put in transfer titles.
	CodeLength = 4
	// CodeChars excludes I and O, which are easily confused with 1 and 0.
	CodeChars = "0123456789ABCDEFGHJKLMNPQRSTUVWXYZ"
)

// GenerateCode returns a random payment code.
func GenerateCode() (string, error) {
	maxVal := big.NewInt(int64(len(CodeChars)))
	var b strings.Builder
	for range CodeLength {
		n, err := rand.Int(rand.Reader, maxVal)
		if err != nil {
			return "", err
		}
		b.WriteByte(CodeChars[n.Int64()])
	}
	return b.String(), nil
}

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]+`)

// FindCode returns the first of codes that appears as a token in a transfer title.
// Customers often type O instead of 0 and I instead of 1, so those are normalized first.
func FindCode(title string, codes []string) string {
	tokens := strings.Fields(nonAlnum.ReplaceAllString(strings.ToUpper(title), " "))
	normalize := strings.NewReplacer("O", "0", "I", "1")
	want := make(map[string]string, len(codes))
	for _, c := range codes {
		want[normalize.Replace(strings.ToUpper(c))] = c
	}
	for _, tok := range tokens {
		if c, ok := want[normalize.Replace(tok)]; ok {
			return c
		}
	}
	return ""
}

// IsValidCode reports whether code contains exactly CodeLength characters from
// CodeChars. Ambiguous letters and non-ASCII lookalikes are rejected.
func IsValidCode(code string) bool {
	if len(code) != CodeLength {
		return false
	}
	for i := 0; i < len(code); i++ {
		c := code[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		if !strings.ContainsRune(CodeChars, rune(c)) {
			return false
		}
	}
	return true
}
