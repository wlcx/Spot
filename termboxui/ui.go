package termboxui

import (
	"github.com/nsf/termbox-go"
)

// Draw a box with optional title
func Drawbox(x, y, w, h int, title string) {

	for i := 0; i < w; i++ {
		termbox.SetCell(x+i, y, '─', termbox.ColorWhite, termbox.ColorDefault)
		termbox.SetCell(x+i, y+h-1, '─', termbox.ColorWhite, termbox.ColorDefault)
	}
	for i := 0; i < h; i++ {
		termbox.SetCell(x, y+i, '│', termbox.ColorWhite, termbox.ColorDefault)
		termbox.SetCell(x+w-1, y+i, '│', termbox.ColorWhite, termbox.ColorDefault)
	}
	if title != "" {
		Print(x+1, y, termbox.ColorWhite, termbox.ColorDefault, "["+title+"]")
	}
	if w > 1 {
		termbox.SetCell(x, y, '┌', termbox.ColorWhite, termbox.ColorDefault)
		termbox.SetCell(x, y+h-1, '└', termbox.ColorWhite, termbox.ColorDefault)
		termbox.SetCell(x+w-1, y, '┐', termbox.ColorWhite, termbox.ColorDefault)
		termbox.SetCell(x+w-1, y+h-1, '┘', termbox.ColorWhite, termbox.ColorDefault)
	}
}

// Draw a bar across row y of the bgcolor bg
func Drawbar(row int, bg termbox.Attribute) {
	x, y := termbox.Size()
	if row < y {
		for i := 0; i < x; i++ {
			termbox.SetCell(i, row, ' ', termbox.ColorWhite, bg)
		}
	}
}
