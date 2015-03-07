package termboxui

import (
	"github.com/nsf/termbox-go"
)

// ListItem is an item in a ScrollList's list. Name is shown in the list, Data is
// for an additional integer value
type ListItem struct {
	Name string
	Data int
}

// ScrollList is a scrollable list of items.
type ScrollList struct {
	Items    []ListItem
	Selected int
	Highlit  int
}

// NewScrollList returns, you guessed it, a new ScrollList instance
func NewScrollList() ScrollList {
	return ScrollList{Highlit: -1}
}

// Draw is called on a ScrollList to draw it on screen. It takes a position for the top
// left corner (x and y) and a width and height (w and h), as well as a focussed bool which
// Changes the color scheme to indicate that the list is focussed on screen
func (l *ScrollList) Draw(x, y, w, h int, focussed bool) {
	// TODO: actually implement scrolling
	if w < 0 || h < 0 {
		return
	}
	for i := 0; i < h; i++ {
		if i == len(l.Items) {
			break
		}
		fgcolor, bgcolor := termbox.ColorWhite, termbox.ColorDefault
		if i == l.Highlit {
			fgcolor = termbox.ColorBlue
		}
		if i == l.Selected { // Use selected colours
			bgcolor = termbox.ColorBlack
			if focussed {
				fgcolor = termbox.ColorYellow
			}
		}
		for ix := x; ix < x+w; ix++ {
			termbox.SetCell(ix, y+i, ' ', fgcolor, bgcolor)
		}
		Printlim(x, y+i, fgcolor, bgcolor, l.Items[i].Name, w)
	}
}

// SelectDown moves the item selection down one
func (l *ScrollList) SelectDown() {
	if l.Selected < len(l.Items)-1 {
		l.Selected++
	}
}

// SelectUp moves the item selection up one
func (l *ScrollList) SelectUp() {
	if l.Selected > 0 {
		l.Selected--
	}
}

// Draw a box with top left corner at x,y height/width h,w and (optional) title title.
// Can also be used to draw lines with a w/h of 1.
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

// Draw a bar across row row of the screen with the background color bg
func Drawbar(row int, bg termbox.Attribute) {
	x, y := termbox.Size()
	if row < y {
		for i := 0; i < x; i++ {
			termbox.SetCell(i, row, ' ', termbox.ColorWhite, bg)
		}
	}
}
