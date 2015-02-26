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

type ListItem struct {
	name string
	data int
}

type ScrollList struct {
	items    []ListItem
	selected int
}

func (l *ScrollList) Draw(x, y, w, h int, focussed bool) {
	// TODO: actually implement scrolling
	for i := 0; i < h; i++ {
		if i == len(l.items) {
			break
		}
		fgcolor, bgcolor := tb.ColorWhite, tb.ColorDefault
		if i == l.selected { // Use selected colours
			bgcolor = tb.ColorBlack
			if focussed {
				fgcolor = tb.ColorYellow
			} else {
				fgcolor = tb.ColorRed
			}
		}
		for ix := x; ix < x+w; ix++ {
			tb.SetCell(ix, y+i, ' ', fgcolor, bgcolor)
		}
		printlim(x, y+i, fgcolor, bgcolor, l.items[i].name, w)
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

type SpotScreen interface {
	Draw(g *spot, x, y int)
	HandleTBEvent(ev tb.Event)
}

type SpotScreenAbout struct{}

func (SpotScreenAbout) Draw(_ *spot, _, _ int) {
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
}

func (SpotScreenAbout) HandleTBEvent(tb.Event) {
}

type SpotScreenPlaylists struct {
	playlists      ScrollList
	tracks         ScrollList
	tracksfocussed bool // if false, playlist list is focussed
}

func (s *SpotScreenPlaylists) Draw(g *spot, x, y int) {
	var playlistlist, tracklist []ListItem
	if g.session.ConnectionState() == sp.ConnectionStateLoggedIn {
		playlistcont, err := g.session.Playlists()
		if err != nil {
			g.cmdline.status = err.Error()
		} else {
			playlistcont.Wait()
			indent := 0
			for i := 0; i < playlistcont.Playlists(); i++ {
				// This is a little fiddly, we have to deal with playlist
				// folders as well as regular playlists
				switch playlistcont.PlaylistType(i) {
				case sp.PlaylistTypePlaylist:
					playlistlist = append(playlistlist, ListItem{strings.Repeat(" ", indent) + playlistcont.Playlist(i).Name(), i})
				case sp.PlaylistTypeStartFolder:
					folder, _ := playlistcont.Folder(i)
					playlistlist = append(playlistlist, ListItem{strings.Repeat(" ", indent) + folder.Name(), i})
					indent++
				case sp.PlaylistTypeEndFolder:
					indent--
				}
			}
			s.playlists.items = playlistlist
			switch playlistcont.PlaylistType(s.playlists.items[s.playlists.selected].data) {
			case sp.PlaylistTypePlaylist:
				playlist := playlistcont.Playlist(s.playlists.items[s.playlists.selected].data)
				playlist.Wait()
				for i := 0; i < playlist.Tracks(); i++ {
					tracklist = append(tracklist, ListItem{playlist.Track(i).Track().Name(), i})
				}
				s.tracks.items = tracklist
			default:
				s.tracks.items = nil
			}
		}
	}
	s.playlists.Draw(0, 1, 30, y-3, !s.tracksfocussed)
	drawbox(30, 1, 1, y-3, "") // Dividing line
	s.tracks.Draw(31, 1, x-31, y-3, s.tracksfocussed)
}

func (s *SpotScreenPlaylists) HandleTBEvent(ev tb.Event) {
	switch ev.Key {
	case tb.KeyTab:
		// Swap focus between playlist and track scrolllists
		s.tracksfocussed = !s.tracksfocussed
	case tb.KeyArrowUp:
		if s.tracksfocussed {
			s.tracks.SelectUp()
		} else {
			s.playlists.SelectUp()
		}
	case tb.KeyArrowDown:
		if s.tracksfocussed {
			s.tracks.SelectDown()
		} else {
			s.playlists.SelectDown()
		}
	}

}

type spot struct {
	session       *sp.Session
	logger        *log.Logger
	cmdline       CmdLine
	quit          bool
	mode          Mode
	currentscreen int
	screens       []SpotScreen
}

func SpotInit(logger *log.Logger, session *sp.Session) spot {
	a := SpotScreenAbout{}
	p := SpotScreenPlaylists{}
	return spot{
		session:       session,
		logger:        logger,
		cmdline:       CmdLine{},
		quit:          false,
		mode:          Normal,
		currentscreen: 0,
		screens:       []SpotScreen{&a, &p},
	}

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

	// Draw active screen
	g.screens[g.currentscreen].Draw(g, x, y)
	// Draw nowplaying
	drawbar(y-2, tb.ColorBlack)
	printtb(0, y-2, tb.ColorBlue, tb.ColorBlack, "Artist - Album - Track")
	// Draw Cmdline
	g.cmdline.Draw()
	tb.Flush()
}

func (g *spot) docommand(cmd string, args []string) string {
	switch cmd {
	case "q", "quit":
		g.quit = true
	case "login":
		if len(args) != 2 {
			return "Usage: :login [username] [password]"
		}
		err := g.session.Login(sp.Credentials{
			Username: args[0],
			Password: args[1],
		}, true)
		if err != nil {
			return "Login Error!"
		}
	case "logout":
		err := g.session.Logout()
		if err != nil {
			return err.Error()
		}
		// If the user issues a logout command, we assume they want to stay
		// logged out
		err = g.session.ForgetMe()
		if err != nil {
			return err.Error()
		}
	case "r", "relogin":
		// TODO: Make this default somehow?
		err := g.session.Relogin()
		if err != nil {
			return err.Error()
		}
	default:
		return "No such command"
	}
	return ""
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
							if result := g.docommand(banana[0], banana[1:]); result != "" {
								g.cmdline.status = result
							}
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
				case tb.KeyTab, tb.KeyArrowUp, tb.KeyArrowDown:
					g.screens[g.currentscreen].HandleTBEvent(ev)
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
							g.currentscreen = 0
						case '1':
							g.currentscreen = 1
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
	s := SpotInit(logger, session)
	s.redraw()
	s.run()
}
