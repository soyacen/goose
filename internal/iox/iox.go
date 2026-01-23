package iox

import "io"

func Len(reader io.Reader) (uint64, bool) {
	switch lener := reader.(type) {
	case interface{ Len() int }:
		return uint64(lener.Len()), true
	case interface{ Len() uint }:
		return uint64(lener.Len()), true
	case interface{ Len() int64 }:
		return uint64(lener.Len()), true
	case interface{ Len() uint64 }:
		return lener.Len(), true
	case interface{ Len() int32 }:
		return uint64(lener.Len()), true
	case interface{ Len() uint32 }:
		return uint64(lener.Len()), true
	case interface{ Len() int16 }:
		return uint64(lener.Len()), true
	case interface{ Len() uint16 }:
		return uint64(lener.Len()), true
	case interface{ Len() int8 }:
		return uint64(lener.Len()), true
	case interface{ Len() uint8 }:
		return uint64(lener.Len()), true

	case interface{ Length() int }:
		return uint64(lener.Length()), true
	case interface{ Length() uint }:
		return uint64(lener.Length()), true
	case interface{ Length() int64 }:
		return uint64(lener.Length()), true
	case interface{ Length() uint64 }:
		return lener.Length(), true
	case interface{ Length() int32 }:
		return uint64(lener.Length()), true
	case interface{ Length() uint32 }:
		return uint64(lener.Length()), true
	case interface{ Length() int16 }:
		return uint64(lener.Length()), true
	case interface{ Length() uint16 }:
		return uint64(lener.Length()), true
	case interface{ Length() int8 }:
		return uint64(lener.Length()), true
	case interface{ Length() uint8 }:
		return uint64(lener.Length()), true

	case interface{ Size() int }:
		return uint64(lener.Size()), true
	case interface{ Size() uint }:
		return uint64(lener.Size()), true
	case interface{ Size() int64 }:
		return uint64(lener.Size()), true
	case interface{ Size() uint64 }:
		return lener.Size(), true
	case interface{ Size() int32 }:
		return uint64(lener.Size()), true
	case interface{ Size() uint32 }:
		return uint64(lener.Size()), true
	case interface{ Size() int16 }:
		return uint64(lener.Size()), true
	case interface{ Size() uint16 }:
		return uint64(lener.Size()), true
	case interface{ Size() int8 }:
		return uint64(lener.Size()), true
	case interface{ Size() uint8 }:
		return uint64(lener.Size()), true
	}
	return 0, false
}
