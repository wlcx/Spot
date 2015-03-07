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
	playlists      *sp.PlaylistContainer
	tracksSL       ui.ScrollList
	playlistsSL    ui.ScrollList
	tracksfocussed bool // if false, playlist list is focussed
	sp             *spot
}

// TODO: some form of asynchronous loading that doesn't block the main thread
// When I tested this (on a slowish connection) there were significant pauses where
// (I assume) the tracks were loading in a playlist.
func NewSpotScreenPlaylists() SpotScreenPlaylists {
	return SpotScreenPlaylists{
		playlistsSL: ui.NewScrollList(),
		tracksSL:    ui.NewScrollList(),
	}
}

func (s *SpotScreenPlaylists) Draw(x, y, w, h int) {
	var err error
	s.playlists, err = s.sp.session.Playlists()
	if err == nil {
		s.playlists.Wait()
	}
	var playlistlist, tracklist []ui.ListItem
	indent := 0
	for i := 0; i < s.playlists.Playlists(); i++ {
		// This is a little fiddly, we have to deal with playlist
		// folders as well as regular playlists
		switch s.playlists.PlaylistType(i) {
		case sp.PlaylistTypePlaylist:
			playlistlist = append(playlistlist, ui.ListItem{strings.Repeat(" ", indent) + s.playlists.Playlist(i).Name(), i})
		case sp.PlaylistTypeStartFolder:
			folder, _ := s.playlists.Folder(i)
			playlistlist = append(playlistlist, ui.ListItem{strings.Repeat(" ", indent) + folder.Name(), i})
			indent++
		case sp.PlaylistTypeEndFolder:
			indent--
		}
	}
	s.playlistsSL.Items = playlistlist
	switch s.playlists.PlaylistType(s.playlistsSL.Items[s.playlistsSL.Selected].Data) {
	case sp.PlaylistTypePlaylist:
		playlist := s.playlists.Playlist(s.playlistsSL.Items[s.playlistsSL.Selected].Data)
		playlist.Wait()
		for i := 0; i < playlist.Tracks(); i++ {
			track := playlist.Track(i).Track()
			track.Wait()
			tracklist = append(tracklist, ui.ListItem{track.Name(), i})
		}
		s.tracksSL.Items = tracklist
	default:
		s.tracksSL.Items = nil
	}
	s.playlistsSL.Draw(x, y, 30, h, !s.tracksfocussed)
	ui.Drawbox(x+30, y, 1, h, "") // Dividing line
	s.tracksSL.Draw(x+31, y, w-31, h, s.tracksfocussed)
}

func (s *SpotScreenPlaylists) HandleTBEvent(ev tb.Event) {
	switch ev.Key {
	case tb.KeyTab:
		// Swap focus between playlist and track scrolllists
		s.tracksfocussed = !s.tracksfocussed
	case tb.KeyArrowUp:
		if s.tracksfocussed {
			s.tracksSL.SelectUp()
		} else {
			s.playlistsSL.SelectUp()
		}
	case tb.KeyArrowDown:
		if s.tracksfocussed {
			s.tracksSL.SelectDown()
		} else {
			s.playlistsSL.SelectDown()
		}
	case tb.KeyEnter:
		if s.tracksfocussed {
			playlistcont, err := s.sp.session.Playlists()
			if err != nil {
				s.sp.cmdline.status = err.Error()
			} else {
				playlistcont.Wait()
				if playlistcont.PlaylistType(s.playlistsSL.Items[s.playlistsSL.Selected].Data) == sp.PlaylistTypePlaylist {
					playlist := playlistcont.Playlist(s.playlistsSL.Items[s.playlistsSL.Selected].Data)
					playlist.Wait()
					if s.tracksSL.Selected < playlist.Tracks() {
						track := playlist.Track(s.tracksSL.Selected).Track()
						track.Wait()
						s.sp.Player.Load(track)
						s.sp.Player.PlayPause()
						s.tracksSL.Highlit = s.tracksSL.Selected
						s.playlistsSL.Highlit = s.playlistsSL.Selected
					}
				}
			}
		}
	}
}
