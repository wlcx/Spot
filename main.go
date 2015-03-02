package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docopt/docopt-go"
	tb "github.com/nsf/termbox-go"
	sp "github.com/op/go-libspotify/spotify"
)

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

type PlayerState int

const (
	Stopped PlayerState = iota
	Playing
	Paused
	Ejected
)

type SpotPlayer struct {
	spplayer  *sp.Player
	track     *sp.Track
	playstate PlayerState
	elapsed   time.Duration
}

func NewSpotPlayer(p *sp.Player) *SpotPlayer {
	return &SpotPlayer{
		spplayer:  p,
		track:     nil,
		playstate: Ejected,
		elapsed:   time.Duration(0),
	}
}

func (p *SpotPlayer) PlayPause() {
	switch p.playstate {
	case Playing:
		p.spplayer.Pause()
		p.playstate = Paused
	case Paused, Stopped:
		p.spplayer.Play()
		p.playstate = Playing
	}
}

func (p *SpotPlayer) Stop() {
	// Pause and seek 0
	switch p.playstate {
	case Playing:
		p.spplayer.Pause()
		fallthrough
	case Paused:
		p.spplayer.Seek(0)
		p.playstate = Stopped
		p.elapsed = time.Duration(0)
	}
}

func (p *SpotPlayer) Load(tr *sp.Track) (err error) {
	if p.playstate != Ejected {
		p.Eject()
	}
	err = p.spplayer.Load(tr)
	if err == nil {
		p.playstate = Stopped
		p.track = tr
	}
	return
}

func (p *SpotPlayer) Eject() {
	p.spplayer.Unload()
	p.track = nil
	p.playstate = Ejected
	p.elapsed = time.Duration(0)
}

func (p *SpotPlayer) NowPlaying() map[string]string {
	var artists string
	for i := 0; i < p.track.Artists(); i++ {
		artists += p.track.Artist(i).Name() + " " //TODO: make this nicer
	}
	return map[string]string{
		"artist":   artists,
		"album":    p.track.Album().Name(),
		"track":    p.track.Name(),
		"duration": PrettyDuration(p.track.Duration()),
		"elapsed":  PrettyDuration(p.elapsed),
	}
}

func (p *SpotPlayer) AddElapsed(dur time.Duration) {
	p.elapsed += dur
}

func (p *SpotPlayer) Seek(pos time.Duration) {
	p.spplayer.Seek(pos)
	p.elapsed = pos
}

type spot struct {
	session       *sp.Session
	logger        *log.Logger
	cmdline       CmdLine
	quit          bool
	mode          Mode
	currentscreen int
	screens       []SpotScreen
	loggedin      bool
	Player        *SpotPlayer
	audiowriter   *audioWriter
}

func SpotInit(logger *log.Logger, session *sp.Session, aw *audioWriter) (sp spot) {
	a := SpotScreenAbout{}
	p := SpotScreenPlaylists{}
	sp = spot{
		session:       session,
		logger:        logger,
		cmdline:       CmdLine{},
		quit:          false,
		mode:          Normal,
		currentscreen: 0,
		screens:       []SpotScreen{&a, &p},
		Player:        NewSpotPlayer(session.Player()),
		audiowriter:   aw,
	}
	p.sp = &sp
	return

}

var PlayerstateSymbols = map[PlayerState]string{
	Playing: ">",
	Stopped: ".",
	Paused:  "|",
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
	var nowplayingstr string
	switch g.Player.playstate {
	case Ejected:
		nowplayingstr = "Nothing playing"
	case Playing, Paused, Stopped:
		np := g.Player.NowPlaying()
		nowplayingstr = fmt.Sprintf("%s %s/%s %s - %s", PlayerstateSymbols[g.Player.playstate], np["elapsed"], np["duration"], np["track"], np["artist"])
	}
	printtb(0, y-2, tb.ColorBlue, tb.ColorBlack, nowplayingstr)
	// Draw Cmdline
	g.cmdline.Draw()
	tb.Flush()
}

func (g *spot) ChangeScreen(index int) {
	if index != 0 && !g.loggedin {
		g.cmdline.status = "Log in to do that"
	} else {
		g.currentscreen = index
	}
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
		g.currentscreen = 0 // Swap to about screen TODO:less magic numbery
		err := g.session.Logout()
		if err != nil {
			return err.Error()
		}
		g.loggedin = false
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
	case "load", "l":
		if !g.loggedin {
			return "Login first!"
		}
		link, err := g.session.ParseLink(args[0])
		if err != nil {
			return err.Error()
		}
		track, err := link.Track()
		if err != nil {
			return err.Error()
		}
		track.Wait()
		if err := g.Player.Load(track); err != nil {
			return err.Error()
		}
		return "Loaded!"
	case "eject", "e":
		g.Player.Eject()
	case "seek", "s":
		if len(args) != 1 {
			return "Usage: seek <seconds>"
		}
		secs, err := strconv.Atoi(args[0])
		if err != nil {
			return "Enter a valid number of seconds"
		}
		g.Player.Seek(time.Duration(secs) * time.Second)
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
					} else {
						g.screens[g.currentscreen].HandleTBEvent(ev)
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
						case 'c':
							g.Player.PlayPause()
						case 'v':
							g.Player.Stop()
						case '0':
							g.ChangeScreen(0)
						case '1':
							g.ChangeScreen(1)
						}
					}
				}

			case tb.EventResize:
				g.redraw()
			}
		case err := <-g.session.LoggedInUpdates():
			if err != nil {
				g.cmdline.status = err.Error()
			} else {
				g.loggedin = true
			}
		case <-g.session.LoggedOutUpdates():
			g.cmdline.status = "Logged out"
			g.loggedin = false
		case <-g.session.ConnectionStateUpdates():
			// Do nothing, we just want to trigger a redraw
		case <-g.session.EndOfTrackUpdates():
			g.Player.Stop() // We use this to Synchronise Player's state
		case time := <-g.audiowriter.ticks:
			g.Player.AddElapsed(time)
		}
		g.redraw()
		if g.quit {
			// Clean up libspotify stuff before we terminate
			g.session.Logout()
			g.session.Close()
			// Send interrupt event and wait for event goroutine to terminate
			tb.Interrupt()
			wg.Wait()
			break
		}
	}
}

func parseArgs() (args map[string]interface{}, err error) {
	usage := `spot

Usage:
	spot
	spot -h | --help
	spot -v | --version

Options:
	-h, --help        Show this help text
	-v, --version     Display spot's version
`
	args, err = docopt.Parse(usage, nil, true, "Spot "+version, false)
	return
}

func main() {
	_, err := parseArgs()
	if err != nil {
		log.Fatalln(err)
	}
	err = tb.Init()
	if err != nil {
		log.Fatal(err)
	}
	defer tb.Close()
	AudioInit()
	defer AudioDeinit()
	aw, _ := NewAudioWriter()
	session, err := sp.NewSession(&sp.Config{
		ApplicationKey:   appkey,
		ApplicationName:  "Spot",
		SettingsLocation: "tmp",
		AudioConsumer:    aw,
	})
	if err != nil {
		log.Fatal(err)
	}
	s := SpotInit(nil, session, aw)
	s.redraw()
	s.run()
}
