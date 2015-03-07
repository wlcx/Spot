package main

import (
	"strings"

	tb "github.com/nsf/termbox-go"
	sp "github.com/op/go-libspotify/spotify"
	ui "github.com/wlcx/spot/termboxui"
)

type SpotScreen interface {
	Draw(x, y, h, w int)
	HandleTBEvent(ev tb.Event)
}

type SpotScreenAbout struct{}

func (SpotScreenAbout) Draw(_, _, w, _ int) {
	ui.Printc(w/2, 5, tb.ColorGreen, tb.ColorDefault, `                     __ `)
	ui.Printc(w/2, 6, tb.ColorGreen, tb.ColorDefault, `   _________  ____  / /_`)
	ui.Printc(w/2, 7, tb.ColorGreen, tb.ColorDefault, `  / ___/ __ \/ __ \/ __/`)
	ui.Printc(w/2, 8, tb.ColorGreen, tb.ColorDefault, ` (__  ) /_/ / /_/ / /_  `)
	ui.Printc(w/2, 9, tb.ColorGreen, tb.ColorDefault, `/____/ .___/\____/\__/  `)
	ui.Printc(w/2, 10, tb.ColorGreen, tb.ColorDefault, `    /_/                 `)
	ui.Printc(w/2, 12, tb.ColorWhite, tb.ColorDefault, "Welcome to Spot "+version)
	ui.Printc(w/2, 13, tb.ColorWhite, tb.ColorDefault, "A simple, fast command line Spotify Client")
	ui.Printc(w/2, 15, tb.ColorWhite, tb.ColorDefault, "Spot uses vim-like commands. Use the source for now.")
}

func (SpotScreenAbout) HandleTBEvent(tb.Event) {
}

type SpotScreenPlaylists struct {
	playlists      ScrollList
	tracks         ScrollList
	tracksfocussed bool // if false, playlist list is focussed
	sp             *spot
}

func (s *SpotScreenPlaylists) Draw(x, y, w, h int) {
	var playlistlist, tracklist []ListItem
	if s.sp.loggedin {
		playlistcont, err := s.sp.session.Playlists()
		if err != nil {
			s.sp.cmdline.status = err.Error()
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
	s.playlists.Draw(x, y, 30, h, !s.tracksfocussed)
	ui.Drawbox(x+30, y, 1, h, "") // Dividing line
	s.tracks.Draw(x+31, y, w-31, h, s.tracksfocussed)
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
	case tb.KeyEnter:
		if s.tracksfocussed {
			playlistcont, err := s.sp.session.Playlists()
			if err != nil {
				s.sp.cmdline.status = err.Error()
			} else {
				playlistcont.Wait()
				if playlistcont.PlaylistType(s.playlists.items[s.playlists.selected].data) == sp.PlaylistTypePlaylist {
					playlist := playlistcont.Playlist(s.playlists.items[s.playlists.selected].data)
					playlist.Wait()
					if s.tracks.selected < playlist.Tracks() {
						track := playlist.Track(s.tracks.selected).Track()
						track.Wait()
						s.sp.Player.Load(track)
						s.sp.Player.PlayPause()
					}
				}
			}
		}
	}
}
