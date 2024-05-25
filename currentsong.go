package mmpd

import "fmt"

type CurrentSong struct {
	PreviousSong *PlaylistEntry
	CurrentSong  *PlaylistEntry
	NextSong     *PlaylistEntry
}

func NewCurrentSong(status *Status, playlist *Playlist) *CurrentSong {
	if playlist == nil {
		return nil
	}

	fmt.Printf("mpd: song=%d songId=%d nextSong=%d nextSongId=%d\n",
		status.Song, status.SongId, status.NextSong, status.NextSongId)

	currentSong := &CurrentSong{}

	playlistLength := len(playlist.Entries)
	if status.SongId > 0 {
		if status.Song < playlistLength {
			if status.Song > 0 {
				currentSong.PreviousSong = playlist.Entries[status.Song-1]
			}
			currentSong.CurrentSong = playlist.Entries[status.Song]
		}
	}
	if status.NextSongId > 0 && status.NextSong < playlistLength {
		currentSong.NextSong = playlist.Entries[status.NextSong]
	}

	return currentSong
}

func (cs *CurrentSong) Equals(other *CurrentSong) bool {
	if other == nil {
		return false
	}

	return cs.PreviousSong.Equals(other.PreviousSong) &&
		cs.CurrentSong.Equals(other.CurrentSong) &&
		cs.NextSong.Equals(other.NextSong)
}
