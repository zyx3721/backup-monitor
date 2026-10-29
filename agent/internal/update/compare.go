package update

import "strings"

// CompareVersion 比较两个版本号并返回 -1、0 或 1。
func CompareVersion(left, right string) int {
	leftCore, leftPre := splitVersion(left)
	rightCore, rightPre := splitVersion(right)
	if result := compareCore(leftCore, rightCore); result != 0 {
		return result
	}
	return comparePrerelease(leftPre, rightPre)
}

func splitVersion(value string) ([]string, string) {
	text := strings.TrimSpace(value)
	text = strings.TrimPrefix(strings.TrimPrefix(text, "v"), "V")
	prerelease := ""
	if index := strings.IndexAny(text, "+"); index >= 0 {
		text = text[:index]
	}
	if index := strings.IndexAny(text, "-"); index >= 0 {
		prerelease = text[index+1:]
		text = text[:index]
	}
	if text == "" {
		return nil, prerelease
	}
	return strings.Split(text, "."), prerelease
}

func compareCore(left, right []string) int {
	length := len(left)
	if len(right) > length {
		length = len(right)
	}
	for index := 0; index < length; index++ {
		leftPart := "0"
		if index < len(left) {
			leftPart = left[index]
		}
		rightPart := "0"
		if index < len(right) {
			rightPart = right[index]
		}
		if result := compareSegment(leftPart, rightPart); result != 0 {
			return result
		}
	}
	return 0
}

func compareSegment(left, right string) int {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if isNumeric(left) && isNumeric(right) {
		left = strings.TrimLeft(left, "0")
		right = strings.TrimLeft(right, "0")
		if left == "" {
			left = "0"
		}
		if right == "" {
			right = "0"
		}
		if len(left) != len(right) {
			if len(left) < len(right) {
				return -1
			}
			return 1
		}
		return strings.Compare(left, right)
	}
	return strings.Compare(left, right)
}

func comparePrerelease(left, right string) int {
	if left == right {
		return 0
	}
	if left == "" {
		return 1
	}
	if right == "" {
		return -1
	}
	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	length := len(leftParts)
	if len(rightParts) > length {
		length = len(rightParts)
	}
	for index := 0; index < length; index++ {
		if index >= len(leftParts) {
			return -1
		}
		if index >= len(rightParts) {
			return 1
		}
		if result := compareIdentifier(leftParts[index], rightParts[index]); result != 0 {
			return result
		}
	}
	return 0
}

func compareIdentifier(left, right string) int {
	leftNumeric := isNumeric(left)
	rightNumeric := isNumeric(right)
	switch {
	case leftNumeric && rightNumeric:
		return compareSegment(left, right)
	case leftNumeric:
		return -1
	case rightNumeric:
		return 1
	default:
		return strings.Compare(left, right)
	}
}

func isNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
