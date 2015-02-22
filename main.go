package main

import (
	"log"
	"os"
	"strings"
	"sync"
)

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
	_, y := tb.Size()
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

type ScrollList struct {
	items    []string
	selected int
}

func (l *ScrollList) Draw(x, y, w, h int) {
	// TODO: actually implement scrolling
	for i := 0; i < h; i++ {
		if i == len(l.items) {
			break
		}
		fgcolor, bgcolor := tb.ColorWhite, tb.ColorDefault
		if i == l.selected { // Use selected colours
			fgcolor, bgcolor = tb.ColorRed, tb.ColorBlack
		}
		for ix := x; ix < x+w-1; ix++ {
			tb.SetCell(ix, y+i, ' ', fgcolor, bgcolor)
		}
		printlim(x, y+i, fgcolor, bgcolor, l.items[i], w)
	}
}

func (l *ScrollList) SelectDown() {
	if l.selected < len(l.items)-1 {
		l.selected++
	}
}

func (l *ScrollList) SelectUp() {
	if l.selected > 0 {
		l.selected--
	}
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

type SpotScreen int

const (
	SpotScreenAbout SpotScreen = iota
	SpotScreenPlaylists
)

type spot struct {
	session       *sp.Session
	cmdline       CmdLine
	quit          bool
	logger        *log.Logger
	mode          Mode
	currentscreen SpotScreen
	playlists     ScrollList
	tracks        ScrollList
}

// (re)Draws the spot UI
func (g *spot) redraw() {
	tb.Clear(tb.ColorWhite, tb.ColorDefault)
	x, y := tb.Size()
	// Draw top bar
	drawbar(0, tb.ColorBlack)
	printtb(0, 0, tb.AttrBold, tb.ColorBlack, "Spot "+version)

	// Get the StatusMsg (message and color) for current spotify session state
	// and print it at the top right
	statusmsg := ConnstateMsg[g.session.ConnectionState()]
	printtbrev(x, 0, statusmsg.Colour, tb.ColorBlack, statusmsg.Msg)

	// Draw screens
	switch g.currentscreen {
	case SpotScreenAbout:
		printtb(8, 5, tb.ColorGreen, tb.ColorDefault, `                     __ `)
		printtb(8, 6, tb.ColorGreen, tb.ColorDefault, `   _________  ____  / /_`)
		printtb(8, 7, tb.ColorGreen, tb.ColorDefault, `  / ___/ __ \/ __ \/ __/`)
		printtb(8, 8, tb.ColorGreen, tb.ColorDefault, ` (__  ) /_/ / /_/ / /_  `)
		printtb(8, 9, tb.ColorGreen, tb.ColorDefault, `/____/ .___/\____/\__/  `)
		printtb(8, 10, tb.ColorGreen, tb.ColorDefault, `    /_/                 `)
		printtb(8, 12, tb.ColorWhite, tb.ColorDefault, "Version "+version)
		printtb(40, 6, tb.ColorWhite, tb.ColorDefault, "Welcome to Spot!")
		printtb(40, 7, tb.ColorWhite, tb.ColorDefault, "A simple, fast command line Spotify Client")
		printtb(40, 9, tb.ColorWhite, tb.ColorDefault, "Login by typing")
		printtb(40, 10, tb.ColorWhite, tb.ColorDefault, ":login <username> <password>")
	case SpotScreenPlaylists:
		var playlistnames, tracknames []string
		if g.session.ConnectionState() == sp.ConnectionStateLoggedIn {
			playlistcont, err := g.session.Playlists()
			if err != nil {
				g.cmdline.status = err.Error()
			} else {
				playlistcont.Wait()
				indent := 0
				for i := 0; i < playlistcont.Playlists(); i++ {
					// This is a little fiddly, we have to deal with playlist
					// folders as well as regular
					// playlistshttp://en.wikipedia.org/wiki/Category:Computing-related_lists
					switch playlistcont.PlaylistType(i) {
					case sp.PlaylistTypePlaylist:
						playlistnames = append(playlistnames, strings.Repeat(" ", indent)+playlistcont.Playlist(i).Name())
					case sp.PlaylistTypeStartFolder:
						folder, _ := playlistcont.Folder(i)
						playlistnames = append(playlistnames, strings.Repeat(" ", indent)+folder.Name())
						indent++
					case sp.PlaylistTypeEndFolder:
						indent--
					}
				}
				g.playlists.items = playlistnames
				switch playlistcont.PlaylistType(g.playlists.selected) {
				case sp.PlaylistTypePlaylist:
					playlist := playlistcont.Playlist(g.playlists.selected)
					playlist.Wait()
					for i := 0; i < playlist.Tracks(); i++ {
						tracknames = append(tracknames, playlist.Track(i).Track().Name())
					}
					g.tracks.items = tracknames
				default:
					g.tracks.items = nil
				}
			}
		}
		drawbox(0, 1, x, y-3, "Playlists")
		g.playlists.Draw(1, 2, 30, y-5)
		g.tracks.Draw(31, 2, x-32, y-5)
	}
	// Draw nowplaying
	drawbar(y-2, tb.ColorBlack)
	printtb(0, y-2, tb.ColorBlue, tb.ColorBlack, "Artist - Album - Track")
	// Draw Cmdline
	g.cmdline.Draw()
	tb.Flush()
}

func (g *spot) docommand(cmd string, args []string) {
	switch cmd {
	case "q", "quit":
		g.quit = true
	case "login":
		if len(args) != 2 {
			g.cmdline.status = "Usage: :login [username] [password]"
			return
		}
		err := g.session.Login(sp.Credentials{
			Username: args[0],
			Password: args[1],
		}, true)
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
	case "r", "relogin":
		// TODO: Make this default somehow?
		err := g.session.Relogin()
		if err != nil {
			g.cmdline.status = err.Error()
		}
	default:
		g.cmdline.status = "No such command"
	}
}

func (g *spot) run() {
	eventCh := make(chan tb.Event)
	wg := new(sync.WaitGroup)
	go func() {
		wg.Add(1)
		for {
			ev := tb.PollEvent()
			if ev.Type == tb.EventInterrupt {
				// We use this as a signal to terminate
				close(eventCh)
				wg.Done()
				break
			}
			eventCh <- ev
		}
	}()

	for {
		// Main run loop. Switch on termbox events (and later stuff from
		// audio?)
		select {
		case ev := <-eventCh:
			switch ev.Type {
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
				case tb.KeyArrowUp:
					switch g.currentscreen {
					case SpotScreenPlaylists:
						g.playlists.SelectUp()
					}
				case tb.KeyArrowDown:
					switch g.currentscreen {
					case SpotScreenPlaylists:
						g.playlists.SelectDown()
					}

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
						case '0':
							g.currentscreen = SpotScreenAbout
						case '1':
							g.currentscreen = SpotScreenPlaylists
						}
					}
				}

			case tb.EventResize:
				g.redraw()
			}
		case err := <-g.session.LoggedInUpdates():
			if err != nil {
				g.cmdline.status = err.Error()
			}
		case <-g.session.LoggedOutUpdates():
			g.cmdline.status = "Logged out"
		case <-g.session.ConnectionStateUpdates():
			// Do nothing, we just want to trigger a redraw
		}
		g.redraw()
		if g.quit {
			// Send interrupt event and wait for event goroutine to terminate
			tb.Interrupt()
			wg.Wait()
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
