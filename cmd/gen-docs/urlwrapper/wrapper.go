package urlwrapper

import (
	"fmt"

	"mvdan.cc/xurls/v2"
)

// MDWrap wraps bare URLs in markdown link syntax while preserving markdown context.
// It avoids double-wrapping URLs that are already in markdown format.
// URLs inside backticks are left as-is since they typically represent literal values.
func MDWrap(text string) string {
	xurlsStrict := xurls.Strict()

	// Build backtick map once at the start
	insideBackticks := buildBacktickMap(text)

	offset := 0
	result := text

	for {
		match := xurlsStrict.FindStringIndex(result[offset:])
		if match == nil {
			break
		}

		startIdx := offset + match[0]
		endIdx := offset + match[1]
		url := result[startIdx:endIdx]

		if shouldWrapURL(result, startIdx, insideBackticks) {
			wrapped := fmt.Sprintf("[%s](%s)", url, url)
			lenDiff := len(wrapped) - len(url)

			// Replace the URL with wrapped version
			result = result[:startIdx] + wrapped + result[endIdx:]

			// Update the backtick map by shifting positions after the insertion
			insideBackticks = shiftBacktickMap(insideBackticks, startIdx, lenDiff)

			// Move offset past the wrapped URL
			offset = startIdx + len(wrapped)
		} else {
			// Move offset past this URL
			offset = endIdx
		}
	}

	return result
}

// shiftBacktickMap adjusts the backtick map positions after text insertion.
// All positions at or after insertPos are shifted by shiftAmount.
func shiftBacktickMap(insideBackticks map[int]bool, insertPos int, shiftAmount int) map[int]bool {
	if shiftAmount == 0 {
		return insideBackticks
	}

	newMap := make(map[int]bool, len(insideBackticks)+shiftAmount)

	for pos, isInside := range insideBackticks {
		if pos < insertPos {
			// Positions before insertion stay the same
			newMap[pos] = isInside
		} else {
			// Positions at or after insertion are shifted
			newMap[pos+shiftAmount] = isInside
		}
	}

	return newMap
}

// shouldWrapURL determines if a URL at the given position should be wrapped in markdown link syntax.
func shouldWrapURL(text string, startIdx int, insideBackticks map[int]bool) bool {
	// Don't wrap if already part of a markdown link: [text](URL)
	if startIdx >= 2 && text[startIdx-2:startIdx] == "](" {
		return false
	}

	// Don't wrap if inside backticks (literal values)
	if insideBackticks[startIdx] {
		return false
	}

	return true
}

// buildBacktickMap creates a map of character positions that are inside backtick code spans.
// It properly handles paired backticks and ignores escaped backticks.
func buildBacktickMap(text string) map[int]bool {
	insideCode := make(map[int]bool)
	i := 0

	for i < len(text) {
		if isEscapedBacktick(text, i) {
			i++
			continue
		}

		if text[i] == '`' {
			backtickCount := countConsecutiveBackticks(text, i)
			startPos := i
			i += backtickCount

			closingPos := findClosingBackticks(text, i, backtickCount)
			if closingPos != -1 {
				markRangeAsCode(insideCode, startPos, closingPos+backtickCount)
				i = closingPos + backtickCount
			}
			continue
		}
		i++
	}

	return insideCode
}

// isEscapedBacktick checks if the backtick at position i is escaped with a backslash.
// It properly handles cases where the backslash itself is escaped (e.g., \\`).
func isEscapedBacktick(text string, i int) bool {
	if i == 0 || text[i] != '`' {
		return false
	}

	// Count consecutive backslashes before the backtick
	backslashCount := 0
	pos := i - 1
	for pos >= 0 && text[pos] == '\\' {
		backslashCount++
		pos--
	}

	// If odd number of backslashes, the backtick is escaped
	// If even number (including 0), the backtick is not escaped
	return backslashCount%2 == 1
}

// countConsecutiveBackticks counts how many backticks appear consecutively starting at position i.
func countConsecutiveBackticks(text string, i int) int {
	count := 0
	for i < len(text) && text[i] == '`' {
		count++
		i++
	}
	return count
}

// markRangeAsCode marks all positions in the given range as being inside code.
func markRangeAsCode(insideCode map[int]bool, start, end int) {
	for pos := start; pos < end; pos++ {
		insideCode[pos] = true
	}
}

// findClosingBackticks finds the position of closing backticks that match the opening count.
// Returns -1 if no matching closing backticks are found.
func findClosingBackticks(text string, startPos int, count int) int {
	i := startPos
	for i < len(text) {
		if isEscapedBacktick(text, i) {
			i++
			continue
		}

		if text[i] == '`' {
			closingCount := countConsecutiveBackticks(text, i)
			if closingCount == count {
				return i
			}
			i += closingCount
			continue
		}
		i++
	}
	return -1
}
