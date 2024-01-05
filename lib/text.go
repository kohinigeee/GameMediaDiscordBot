package lib

import (
	"fmt"
	"strings"
)

func IsURL(str string) bool {
	prefixs := []string{"http://", "https://"}
	result := false

	for _, prefix := range prefixs {
		if strings.HasPrefix(str, prefix) {
			result = true
		}
	}

	return result
}

func ConvertNotShowLink(content string) string {
	lines := strings.Split(content, "\n")

	result := ""
	for line := range lines {
		tokens := strings.Split(lines[line], " ")

		for _, token := range tokens {
			if IsURL(token) {
				result += fmt.Sprintf("<%s> ", token)
			} else {
				result += token + " "
			}
		}
		result := strings.TrimRight(result, " ")
		result += "\n"
	}

	return result
}
