package util

import "strings"

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

// Pluralize Pluralize word
func Pluralize(number int, variations [3]string) string {
	cases := []int{2, 0, 1, 1, 1, 2}
	var index int
	if number%100 > 4 && number%100 < 20 || number%10 >= 5 {
		index = 2
	} else {
		index = cases[number%10]
	}

	return variations[index]
}

// FixMarkdown Abrupt Telegram markdown fix
func FixMarkdown(text string) string {
	text = strings.TrimRight(text, "`")
	tripleCnt := strings.Count(text, "```")
	singleCnt := strings.Count(strings.Replace(text, "```", "", -1), "`")

	if tripleCnt%2 == 1 {
		text += "```"
	}
	if singleCnt%2 == 1 {
		text += "`"
	}

	return text
}
