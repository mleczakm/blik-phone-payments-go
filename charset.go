package payments

import (
	"encoding/base64"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

var base64Std = base64.StdEncoding

// toUTF8 converts the Polish legacy charsets banks still use to UTF-8.
func toUTF8(charset, s string) string {
	var dec interface{ String(string) (string, error) }
	switch strings.ToLower(charset) {
	case "iso-8859-2", "latin2":
		dec = charmap.ISO8859_2.NewDecoder()
	case "windows-1250", "cp1250":
		dec = charmap.Windows1250.NewDecoder()
	default:
		return s
	}
	if out, err := dec.String(s); err == nil {
		return out
	}
	return s
}
