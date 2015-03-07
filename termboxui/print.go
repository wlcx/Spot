package termboxui

import (
	"unicode/utf8"

	"github.com/nsf/termbox-go"
)

// Print sets a line of cells starting at x,y to the string msg
func Print(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, aRune := range msg {
		termbox.SetCell(x, y, aRune, fg, bg)
		x++
	}
}

// Printr sets a line of cells ending at x, y to the string msg.
// Useful for right-aligning text.
func Printr(x, y int, fg, bg termbox.Attribute, msg string) {
	i := utf8.RuneCountInString(msg)
	for _, aRune := range msg {
		termbox.SetCell(x-i, y, aRune, fg, bg)
		i--
	}
}

// Printc sets a line of cells centered around x, y to the string msg.
// Useful for center-aligning text.
func Printc(x, y int, fg, bg termbox.Attribute, msg string) {
	offset := utf8.RuneCountInString(msg) / 2
	for pos, aRune := range msg {
		termbox.SetCell(x-offset+pos, y, aRune, fg, bg)
	}
}

// Printlim is a wrapper around Print which limits the length of the printed message
// to lim, replacing the last two runes with "..". Useful for columnular layouts.
func Printlim(x, y int, fg, bg termbox.Attribute, msg string, lim int) {
	if utf8.RuneCountInString(msg) > lim {
		msg = msg[:lim-2] + ".."
	}
	Print(x, y, fg, bg, msg)
}
