package strconvx

import (
	"strconv"
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
