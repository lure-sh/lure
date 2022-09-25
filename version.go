package main

import (
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

// vercmp compares two version strings.
// It returns 1 if v1 is greater,
// 0 if the versions are equal,
// and -1 if v2 is greater
func vercmp(v1, v2 string) int {
	if v1 == v2 {
		return 0
	}

	return sepVerCmp(sepLabel(v1), sepLabel(v2))
}

func sepVerCmp(e1, e2 []string) int {
	if slices.Equal(e1, e2) {
		return 0
	}

	// proc stores the amount of elements processed
	proc := 0

	for i := 0; i < len(e1); i++ {
		proc++

		if i >= len(e2) {
			return 1
		}

		elem1 := e1[i]
		elem2 := e2[i]

		if elem1 == elem2 {
			continue
		}

		if isNumElem(elem1) && isNumElem(elem2) {
			elem1v, err := strconv.ParseInt(elem1, 10, 64)
			if err != nil {
				// error should never happen due to isNumElem()
				panic(err)
			}

			elem2v, err := strconv.ParseInt(elem2, 10, 64)
			if err != nil {
				// error should never happen due to isNumElem()
				panic(err)
			}

			if elem1v > elem2v {
				return 1
			} else if elem1v < elem2v {
				return -1
			}
		} else if isNumElem(elem1) && isAlphaElem(elem2) {
			return 1
		} else if isAlphaElem(elem1) && isNumElem(elem2) {
			return -1
		} else if isAlphaElem(elem1) && isAlphaElem(elem2) {
			if elem1 > elem2 {
				return 1
			} else if elem1 < elem2 {
				return -1
			}
		}
	}

	if proc < len(e2) {
		return -1
	}

	return 0
}

func sepLabel(label string) []string {
	const (
		other = iota
		alpha
		num
	)

	var (
		curType uint8
		out     []string
		sb      strings.Builder
	)

	for _, char := range label {
		if isNum(char) {
			if curType != num && curType != other {
				out = append(out, sb.String())
				sb.Reset()
			}

			sb.WriteRune(char)
			curType = num
		} else if isAlpha(char) {
			if curType != alpha && curType != other {
				out = append(out, sb.String())
				sb.Reset()
			}

			sb.WriteRune(char)
			curType = alpha
		} else {
			if curType != other {
				out = append(out, sb.String())
				sb.Reset()
			}
			curType = other
		}
	}

	if sb.Len() != 0 {
		out = append(out, sb.String())
	}

	return out
}

func isNumElem(s string) bool {
	// Check only the first rune as all elements
	// should consist of the same type of rune
	return isNum([]rune(s[:1])[0])
}

func isNum(r rune) bool {
	return r >= '0' && r <= '9'
}

func isAlphaElem(s string) bool {
	// Check only the first rune as all elements
	// should consist of the same type of rune
	return isAlpha([]rune(s[:1])[0])
}

func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}
