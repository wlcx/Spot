package main

import "os"
import "log"
import "strings"

import sp "github.com/op/go-libspotify/spotify"
import tb "github.com/nsf/termbox-go"

// This should be injected at compile time by a script
var version string

type CmdLine struct {
	Text    []rune
	history [][]rune
	status  string
}

func (c *CmdLine) Draw() {
	x, y := tb.Size()
	// First clear entire row
	for i := 0; i < x; i++ {
		tb.SetCell(i, y-1, ' ', tb.ColorWhite, tb.ColorDefault)
	}
	// If there is a status message, draw it, otherwise draw the current
	// command in c.Text
	if c.status != "" {
		printtb(0, y-1, tb.ColorRed, tb.ColorDefault, c.status)
	} else {
		for i, r := range c.Text {
			tb.SetCell(i, y-1, r, tb.ColorWhite, tb.ColorDefault)
		}
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

// Should be run after command has. Empty current command buffer and push it to
// history
func (c *CmdLine) Push() {
	c.history = append(c.history, c.Text)
	c.Text = nil
}

func (c *CmdLine) Clear() {
	c.Text = nil
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

type spot struct {
	session   *sp.Session
	cmdline   CmdLine
	quit      bool
	logger    *log.Logger
	mode      Mode
}

func (g *spot) redraw() {
	tb.Clear(tb.ColorWhite, tb.ColorDefault)
	x, y := tb.Size()
	// Draw top bar
	for i := 0; i <= x; i++ {
		tb.SetCell(i, 0, ' ', tb.ColorDefault, tb.ColorBlack)
	}
	printtb(0, 0, tb.AttrBold, tb.ColorBlack, "Spot "+version)

	// Get the StatusMsg (message and color) for current spotify session state
	// and print it at the top right
	statusmsg := ConnstateMsg[g.session.ConnectionState()]
	printtbrev(x, 0, statusmsg.Colour, tb.ColorBlack, statusmsg.Msg)

	// Draw screen
	drawbox(0, 1, x, y-2, "Playlists")

	//Draw Cmdline
	g.cmdline.Draw()
	tb.Flush()
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

func (g *spot) docommand(cmd string, args []string) {
	switch cmd {
	case "q", "quit":
		g.quit = true
	case "login":
		if len(args) != 2 {
			g.cmdline.status = "Usage: :login <username> <password>"
			return
		}
		err := g.session.Login(sp.Credentials{
			Username: args[0],
			Password: args[1],
		}, false) // Don't remember for now, TODO: this
		if err != nil {
			g.cmdline.status = "Login Error!"
		}
	case "logout":
		err := g.session.Logout()
		if err != nil {
			g.cmdline.status = err.Error()
		}
		// If the user issues a logout command, we assume they want to stay
		// logged out
		err = g.session.ForgetMe()
		if err != nil {
			g.cmdline.status = err.Error()
		}
	default:
		g.cmdline.status = "No such command"
	}
}

func (g *spot) run() {
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
						banana := strings.Split(string(g.cmdline.Text[1:]), " ")
						g.docommand(banana[0], banana[1:])
						g.cmdline.Push()
					}
				}
			case tb.KeyBackspace, tb.KeyBackspace2:
				if g.mode == Command {
					g.cmdline.DelChar()
				}
			case tb.KeyDelete:
				if g.mode == Command {
					// TODO: this, requires a cursor
				}
			case tb.KeySpace:
				if g.mode == Command {
					g.cmdline.AddChar(' ')
				}
			case tb.KeyEsc:
				if g.mode == Command {
					g.cmdline.Clear()
				}
				g.mode = Normal

			default:
				if g.mode == Command && ev.Ch != 0 {
					g.cmdline.AddChar(ev.Ch)
				} else {
					// run keybinding TODO: make more configurable
					switch ev.Ch {
					case ':':
						g.mode = Command
						g.cmdline.status = ""
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
			// Clean up libspotify stuff before we terminate
			g.session.Logout()
			g.session.Close()
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
		ApplicationName:  "Spot",
		SettingsLocation: "tmp",
		AudioConsumer:    nil,
	})
	if err != nil {
		logger.Fatal(err)
	}
	gs := spot{
		logger:  logger,
		quit:    false,
		session: session,
		cmdline: CmdLine{},
		mode:    Normal,
	}
	gs.redraw()
	gs.run()
}
