package main

import "fmt"
import "os"
import "log"

import sp "github.com/op/go-libspotify/spotify"
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

type CmdLine struct {
	Text    []rune
	history [][]rune
}

func (c *CmdLine) Draw() {
	x, y := tb.Size()
	//Print our command line to the bottom of the window
	for i, r := range c.Text {
		tb.SetCell(i, y-1, r, tb.ColorWhite, tb.ColorDefault)
	}
	for i := len(c.Text); i < x; i++ {
		tb.SetCell(i, y-1, ' ', tb.ColorWhite, tb.ColorDefault)
	}
}

func (c *CmdLine) AddChar(r rune) {
	//TODO cursor
	c.Text = append(c.Text, r)
}

func (c *CmdLine) DelChar() {
	//TODO cursor, delete key
	if len(c.Text) > 0 {
		c.Text = c.Text[:len(c.Text)-1]
	}
}

// Draw a status message over the commandbar.
// Persists until next screen redraw
func drawstatus(msg string) {
	x, y := tb.Size()
	for i := 0; i < x; i++ {
		tb.SetCell(i, y-1, ' ', tb.ColorWhite, tb.ColorBlack)
	}
	printtb(0, y-1, tb.ColorRed, tb.ColorBlack, msg)
	tb.Flush()
}

func redraw() {
	tb.Flush()
	x, _ := tb.Size()
	// Draw top bar
	for i := 0; i <= x; i++ {
		tb.SetCell(i, 0, ' ', tb.ColorDefault, tb.ColorBlack)
	}
	printtb(0, 0, tb.AttrBold, tb.ColorBlack, "GSPOT v0.0.1")
	printtbrev(x, 0, tb.ColorGreen, tb.ColorBlack, "Online")
	// Draw bottom bar

	//Draw Cmdline
	cmdline.Draw()
	tb.Flush()
}

func docommand(cmd []rune) {
	if cmd[0] == 'q' {
		quit = true
	}
}

var quit = false
var cmdline CmdLine

func main() {
	logger := log.New(os.Stdout, "[>] ", log.Lshortfile)
	err := tb.Init()
	if err != nil {
		logger.Panic(err)
	}
	defer tb.Close()
	commandmode := false
	redraw()
	drawstatus("Welcome")

	for {
		// Main run loop. Switch on termbox events (and later stuff from
		// audio?)
		switch ev := tb.PollEvent(); ev.Type {
		case tb.EventKey:
			switch ev.Key {
			case tb.KeyEnter:
				if commandmode { // Finish command
					commandmode = false
					if len(cmdline.Text) > 1 {
						docommand(cmdline.Text[1:])
					}
				}
			case tb.KeyBackspace, tb.KeyBackspace2:
				if commandmode {
					cmdline.DelChar()
				}
			default:
				if commandmode && ev.Ch != 0 {
					cmdline.AddChar(ev.Ch)
				} else {
					// run keybinding TODO: make more configurable
					switch ev.Ch {
					case ':':
						commandmode = true
						cmdline.AddChar(':')
					case 'q':
						//Quit
						quit = true
					}
				}
			}

		case tb.EventResize:
			redraw()
		}
		redraw()
		if quit {
			break
		}
	}
}

func spotify() {
	conf := sp.Config{
		appkey,
		"Gspot",
		"",
		"",
		"",
		false,
		true,
		true,
		nil,
	}
	_, err := sp.NewSession(&conf)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
