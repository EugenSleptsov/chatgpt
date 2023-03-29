package util

// Title Make first letter capital
func Title(s string) string {
	if len(s) == 0 {
		return s
	}

	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] = r[0] - 'a' + 'A'
	}

	return string(r)
}
