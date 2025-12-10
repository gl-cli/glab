package utils

import (
	"regexp"
	"strconv"
	"strings"
)

// ReplaceNonAlphaNumericChars: Replaces non alpha-numeric values with provided char/string
func ReplaceNonAlphaNumericChars(words, replaceWith string) string {
	reg := regexp.MustCompile("[^A-Za-z0-9]+")
	newStr := reg.ReplaceAllString(strings.Trim(words, " "), replaceWith)
	return newStr
}

func StringToInt(str string) int {
	strInt, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return strInt
}
