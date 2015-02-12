package main

import "os"
import "log"

import sp "github.com/op/go-libspotify/spotify"
import tb "github.com/nsf/termbox-go"

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

// A status message and Termbox color attribute. For display in the top right
type StatusMsg struct {
	Msg    string
	Colour tb.Attribute
}

// Maps between spotify connectionstates to Statusmsg structs
var ConnstateMsg = map[sp.ConnectionState]StatusMsg{
	sp.ConnectionStateLoggedOut:    StatusMsg{"Logged Out", tb.ColorRed},
	sp.ConnectionStateLoggedIn:     StatusMsg{"Logged In", tb.ColorGreen},
	sp.ConnectionStateDisconnected: StatusMsg{"Disconnected", tb.ColorRed},
	sp.ConnectionStateUndefined:    StatusMsg{"???", tb.ColorWhite},
	sp.ConnectionStateOffline:      StatusMsg{"Offline", tb.ColorRed},
}

type Mode int

const (
	Normal Mode = iota
	Command
	Search
)

type gspot struct {
	session *sp.Session
	cmdline CmdLine
	quit    bool
	logger  *log.Logger
	mode    Mode
}

func (g *gspot) redraw() {
	tb.Flush()
	x, _ := tb.Size()
	// Draw top bar
	for i := 0; i <= x; i++ {
		tb.SetCell(i, 0, ' ', tb.ColorDefault, tb.ColorBlack)
	}
	printtb(0, 0, tb.AttrBold, tb.ColorBlack, "GSPOT v0.0.1")

	// Get the StatusMsg (message and color) for current spotify session state
	// and print it at the top right
	statusmsg := ConnstateMsg[g.session.ConnectionState()]
	printtbrev(x, 0, statusmsg.Colour, tb.ColorBlack, statusmsg.Msg)

	//Draw Cmdline
	g.cmdline.Draw()
	tb.Flush()
}

func (g *gspot) docommand(cmd []rune) {
	if cmd[0] == 'q' {
		g.quit = true
	}
}

func (g *gspot) run() {
	for {
		// Main run loop. Switch on termbox events (and later stuff from
		// audio?)
		switch ev := tb.PollEvent(); ev.Type {
		case tb.EventKey:
			switch ev.Key {
			case tb.KeyEnter:
				if g.mode == Command { // Finish command
					g.mode = Normal
					if len(g.cmdline.Text) > 1 {
						g.docommand(g.cmdline.Text[1:])
					}
				}
			case tb.KeyBackspace, tb.KeyBackspace2:
				if g.mode == Command {
					g.cmdline.DelChar()
				}
			default:
				if g.mode == Command && ev.Ch != 0 {
					g.cmdline.AddChar(ev.Ch)
				} else {
					// run keybinding TODO: make more configurable
					switch ev.Ch {
					case ':':
						g.mode = Command
						g.cmdline.AddChar(':')
					case 'q':
						//Quit
						g.quit = true
					}
				}
			}

		case tb.EventResize:
			g.redraw()
		}
		g.redraw()
		if g.quit {
			break
		}
	}
}

func main() {
	logger := log.New(os.Stdout, "[>] ", log.Lshortfile)
	err := tb.Init()
	if err != nil {
		logger.Panic(err)
	}
	defer tb.Close()
	session, err := sp.NewSession(&sp.Config{
		ApplicationKey:   appkey,
		ApplicationName:  "GSpot",
		SettingsLocation: "tmp",
		AudioConsumer:    nil,
	})
	if err != nil {
		logger.Fatal(err)
	}
	gs := gspot{
		logger:  logger,
		quit:    false,
		session: session,
		cmdline: CmdLine{},
		mode:    Normal,
	}
	gs.redraw()
	gs.run()
}
