package mmpd

import (
	"fmt"

	"github.com/fhs/gompd/v2/mpd"
)

func GetCurrentSongs(client *mpd.Client, status *Status) (prevSong, currentSong, nextSong *PlaylistEntry, err error) {
	if status.Song > 0 {
		if status.NextSong > 0 {
			if playlistInfo, err := client.PlaylistInfo(status.Song-1, status.NextSong+1); err != nil {
				return nil, nil, nil,
					fmt.Errorf("mpd: playlistinfo(%d, %d) command failed: %w", status.Song-1, status.NextSong, err)
			} else if len(playlistInfo) != 3 {
				return nil, nil, nil,
					fmt.Errorf("mpd: playlistinfo(%d, %d) returned unexpected data: %+v", status.Song-1, status.NextSong, playlistInfo)
			} else {
				prevSong = ParsePlaylistEntryAttrs(playlistInfo[0])
				currentSong = ParsePlaylistEntryAttrs(playlistInfo[1])
				nextSong = ParsePlaylistEntryAttrs(playlistInfo[2])
			}
		} else {
			if playlistInfo, err := client.PlaylistInfo(status.Song-1, status.Song+1); err != nil {
				return nil, nil, nil,
					fmt.Errorf("mpd: playlistinfo(%d, %d) command failed: %w", status.Song-1, status.Song, err)
			} else if len(playlistInfo) != 2 {
				return nil, nil, nil,
					fmt.Errorf("mpd: playlistinfo(%d, %d) returned unexpected data: %+v", status.Song-1, status.Song, playlistInfo)
			} else {
				prevSong = ParsePlaylistEntryAttrs(playlistInfo[0])
				currentSong = ParsePlaylistEntryAttrs(playlistInfo[1])
				nextSong = nil
			}
		}
	} else if status.NextSong > 0 {
		if playlistInfo, err := client.PlaylistInfo(status.Song, status.NextSong+1); err != nil {
			return nil, nil, nil,
				fmt.Errorf("mpd: playlistinfo(%d, %d) command failed: %w", status.Song, status.NextSong, err)
		} else if len(playlistInfo) != 2 {
			return nil, nil, nil,
				fmt.Errorf("mpd: playlistinfo(%d, %d) returned unexpected data: %+v", status.Song, status.NextSong, playlistInfo)
		} else {
			prevSong = nil
			currentSong = ParsePlaylistEntryAttrs(playlistInfo[0])
			nextSong = ParsePlaylistEntryAttrs(playlistInfo[1])
		}
	} else if status.Song >= 0 {
		if playlistInfo, err := client.PlaylistInfo(status.Song, -1); err != nil {
			return nil, nil, nil,
				fmt.Errorf("mpd: playlistinfo(%d, %d) command failed: %w", status.Song, -1, err)
		} else if len(playlistInfo) != 1 {
			return nil, nil, nil,
				fmt.Errorf("mpd: playlistinfo(%d, %d) returned unexpected data: %+v", status.Song, -1, playlistInfo)
		} else {
			prevSong = nil
			currentSong = ParsePlaylistEntryAttrs(playlistInfo[0])
			nextSong = nil
		}
	}
	return prevSong, currentSong, nextSong, nil
}
