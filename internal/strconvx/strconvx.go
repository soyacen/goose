package strconvx

import (
	"go/token"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/exp/constraints"
)

// StringToBytes converts string to byte slice without a memory allocation.
// For more details, see https://github.com/golang/go/issues/53003#issuecomment-1140276077.
func StringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// BytesToString converts byte slice to string without a memory allocation.
// For more details, see https://github.com/golang/go/issues/53003#issuecomment-1140276077.
func BytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func FormatUint[Unsigned constraints.Unsigned](i Unsigned, base int) string {
	return strconv.FormatUint(uint64(i), base)
}

// GoSanitized converts a string to a valid Go identifier.
func GoSanitized(s string) string {
	// Sanitize the input to the set of valid characters,
	// which must be '_' or be in the Unicode L or N categories.
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, s)

	// Prepend '_' in the event of a Go keyword conflict or if
	// the identifier is invalid (does not start in the Unicode L category).
	r, _ := utf8.DecodeRuneInString(s)
	if token.Lookup(s).IsKeyword() || !unicode.IsLetter(r) {
		return "_" + s
	}
	return s
}
