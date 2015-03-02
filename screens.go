package main

import (
	"strings"

	tb "github.com/nsf/termbox-go"
	sp "github.com/op/go-libspotify/spotify"
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
	sp             *spot
}

func (s *SpotScreenPlaylists) Draw(g *spot, x, y int) {
	var playlistlist, tracklist []ListItem
	if g.loggedin {
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
