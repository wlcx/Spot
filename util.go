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

// Wraps printtb and limits the length of the printed line. Useful for
// column-based layouts
func printlim(x, y int, fg, bg tb.Attribute, msg string, lim int) {
	if len(msg) > lim {
		msg = msg[:lim-4] + "..."
	}
	printtb(x, y, fg, bg, msg)
}

// Draw a box with optional title
func drawbox(x, y, w, h int, title string) {
	tb.SetCell(x, y, '┌', tb.ColorWhite, tb.ColorDefault)
	tb.SetCell(x, y+h-1, '└', tb.ColorWhite, tb.ColorDefault)
	tb.SetCell(x+w-1, y, '┐', tb.ColorWhite, tb.ColorDefault)
	tb.SetCell(x+w-1, y+h-1, '┘', tb.ColorWhite, tb.ColorDefault)
	for i := 1; i < w-1; i++ {
		tb.SetCell(x+i, y, '─', tb.ColorWhite, tb.ColorDefault)
		tb.SetCell(x+i, y+h-1, '─', tb.ColorWhite, tb.ColorDefault)
	}
	for i := 1; i < h-1; i++ {
		tb.SetCell(x, y+i, '│', tb.ColorWhite, tb.ColorDefault)
		tb.SetCell(x+w-1, y+i, '│', tb.ColorWhite, tb.ColorDefault)
	}
	printtb(x+1, y, tb.ColorWhite, tb.ColorDefault, "["+title+"]")
}

// Draw a bar across row y of the bgcolor bg
func drawbar(row int, bg tb.Attribute) {
	x, y := tb.Size()
	if row < y {
		for i := 0; i < x; i++ {
			tb.SetCell(i, row, ' ', tb.ColorWhite, bg)
		}
	}
}
