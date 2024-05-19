package mmpd

import "strings"

type OffOnOneshot string

const Off OffOnOneshot = "0"
const On OffOnOneshot = "1"
const Oneshot OffOnOneshot = "oneshot"

func ParseOffOnOneshot(s string) OffOnOneshot {
	switch strings.ToLower(s) {
	case "0":
		return Off
	case "1":
		return On
	case "oneshot":
		return Oneshot
	default:
		return ""
	}
}
