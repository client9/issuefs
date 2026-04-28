package issue

import "strings"

const slugMaxLen = 60

func Slug(title string) string {
	var b strings.Builder
	b.Grow(len(title))
	prevDash := true
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	s := strings.Trim(b.String(), "-")

	if len(s) <= slugMaxLen {
		return s
	}
	cut := s[:slugMaxLen]
	if i := strings.LastIndexByte(cut, '-'); i > 0 {
		cut = cut[:i]
	}
	return strings.Trim(cut, "-")
}
