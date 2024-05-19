package mmpd

import "github.com/fhs/gompd/v2/mpd"

func GetCurrentSongs(client *mpd.Client, status *Status) (prevSong, currentSong, nextSong *PlaylistEntry, err error) {
	if status.SongId > 0 {
		if status.NextSongId > 0 {
			if playlistInfo, err := client.PlaylistInfo(status.SongId-1, status.NextSongId); err != nil {
				return nil, nil, nil, err
			} else {
				prevSong = ParsePlaylistEntryAttrs(playlistInfo[0])
				currentSong = ParsePlaylistEntryAttrs(playlistInfo[1])
				nextSong = ParsePlaylistEntryAttrs(playlistInfo[2])
			}
		} else {
			if playlistInfo, err := client.PlaylistInfo(status.SongId-1, status.SongId); err != nil {
				return nil, nil, nil, err
			} else {
				prevSong = ParsePlaylistEntryAttrs(playlistInfo[0])
				currentSong = ParsePlaylistEntryAttrs(playlistInfo[1])
				nextSong = nil
			}
		}
	} else if status.NextSongId > 0 {
		if playlistInfo, err := client.PlaylistInfo(status.SongId, status.NextSongId); err != nil {
			return nil, nil, nil, err
		} else {
			prevSong = nil
			currentSong = ParsePlaylistEntryAttrs(playlistInfo[0])
			nextSong = ParsePlaylistEntryAttrs(playlistInfo[1])
		}
	} else {
		if playlistInfo, err := client.PlaylistInfo(status.SongId, -1); err != nil {
			return nil, nil, nil, err
		} else {
			prevSong = nil
			currentSong = ParsePlaylistEntryAttrs(playlistInfo[0])
			nextSong = nil
		}
	}
	return prevSong, currentSong, nextSong, nil
}
