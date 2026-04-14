package cmd

import "strconv"

// parseInt parses a string to int, returning def on error.
// Shared helper for all command files.
func parseInt(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
