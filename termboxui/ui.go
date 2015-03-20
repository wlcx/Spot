package termboxui

import (
	"github.com/nsf/termbox-go"
)

// ListItem is an item in a ScrollList's list. TextL and TextR are displayed in the list,
// aligned to the left and right respectively, and Data is an optional integer
type ListItem struct {
	TextL    string
	TextR    string
	Data     int
	Disabled bool
}

// ScrollList is a scrollable list of items.
type ScrollList struct {
	Items    []ListItem
	Selected int
	Highlit  int
	offset   int
}

// NewScrollList returns, you guessed it, a new ScrollList instance
func NewScrollList() ScrollList {
	return ScrollList{Highlit: -1}
}

// Draw is called on a ScrollList to draw it on screen. It takes a position for the top
// left corner (x and y) and a width and height (w and h), as well as a focussed bool which
// Changes the color scheme to indicate that the list is focussed on screen
func (l *ScrollList) Draw(x, y, w, h int, focussed bool) {
	// The no. of lines kept in view above/below selection when scrolling up/down
	scrollpadding := 2
	if w < 0 || h < 0 {
		return
	}
	//Recalculate offset to keep selection in view
	switch {
	case l.Selected >= (h+l.offset)-scrollpadding-1 && l.offset+h < len(l.Items):
		l.offset += (l.Selected - ((h + l.offset) - scrollpadding - 1))
	case l.Selected <= l.offset+scrollpadding && l.offset > 0:
		l.offset -= (l.offset - l.Selected) + scrollpadding
	}
	displayed := l.Items[l.offset:]
	for i, tr := range displayed {
		if i == h {
			break
		}
		index := i + l.offset
		fgcolor, bgcolor := termbox.ColorWhite, termbox.ColorDefault
		if tr.Disabled {
			fgcolor = termbox.ColorDefault
		} else {
			if index == l.Selected {
				bgcolor = termbox.ColorBlack
				if focussed {
					fgcolor = termbox.ColorYellow
				}
			}
			if index == l.Highlit {
				fgcolor = termbox.ColorBlue
			}
		}
		Drawbar(x, y+i, w, bgcolor)
		Printlim(x, y+i, fgcolor, bgcolor, tr.TextL, w)
		Printr(x+w, y+i, fgcolor, bgcolor, tr.TextR)
	}
}

// SelectDown moves the item selection down one, skipping any disabled items
func (l *ScrollList) SelectDown() {
	for {
		if l.Selected >= len(l.Items)-1 { // Already at (or past!) the bottom
			l.Selected = len(l.Items) - 1 // Reset it just in case (eg list items removed)
			break
		}
		l.Selected++
		if !l.Items[l.Selected].Disabled { // If current item is disabled we loop again
			break
		}
	}
}

// SelectUp moves the item selection up one, skipping any disabled items
func (l *ScrollList) SelectUp() {
	for {
		if l.Selected == 0 { // Already at the top
			break
		}
		l.Selected--
		if !l.Items[l.Selected].Disabled { // If current item is disabled we loop again
			break
		}
	}
}

// Clear clears a ScrollList and resets selection/highlit
func (l *ScrollList) Clear() {
	l.Items = nil
	l.Selected = 0
	l.Highlit = -1
	l.offset = 0
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

// Draw a bar across w columns of row y of the screen starting at col x
// with the background color bg
func Drawbar(x, y, w int, bg termbox.Attribute) {
	for i := x; i < x+w; i++ {
		termbox.SetCell(i, y, ' ', termbox.ColorWhite, bg)
	}
}
