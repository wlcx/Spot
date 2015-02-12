package main

import tb "github.com/nsf/termbox-go"

func printtb(x, y int, fg, bg tb.Attribute, msg string) {
	for _, c := range msg {
		tb.SetCell(x, y, c, fg, bg)
		x++
	}
}

// Prints in row y with last char of msg at x.
// Useful for right-aligning
func printtbrev(x, y int, fg, bg tb.Attribute, msg string) {
	i := len(msg)
	for _, c := range msg {
		tb.SetCell(x-i, y, c, fg, bg)
		i--
	}
}
