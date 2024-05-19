package mmpd

import (
	"strconv"

	"github.com/fhs/gompd/v2/mpd"
)

type Status struct {
	// the name of the current partition (see Partition commands)
	Partition string

	// 0-100 (deprecated: -1 if the volume cannot be determined)
	Volume int

	// 0 or 1
	Repeat bool

	// 0 or 1
	Random bool

	// 0, 1, or oneshot
	Single OffOnOneshot

	// 0, 1 or oneshot 13
	Consume OffOnOneshot

	// 31-bit unsigned integer, the playlist version number
	Playlist uint32

	// integer, the length of the playlist
	PlaylistLength int

	// play, stop, or pause
	State PlayerState

	// playlist song number of the current song stopped on or playing
	Song int

	// playlist songid of the current song stopped on or playing
	SongId int

	// playlist song number of the next song to be played
	NextSong int

	// playlist songid of the next song to be played
	NextSongId int

	// total time elapsed (of current playing/paused song) in seconds (deprecated, use elapsed instead)
	Time int

	// Total time elapsed within the current song in seconds, but with higher resolution.
	Elapsed float32

	// Duration of the current song in seconds.
	Duration int

	// instantaneous bitrate in kbps
	Bitrate int

	// crossfade in seconds (see Cross-Fading)
	CrossFade int

	// mixramp threshold in dB
	MixRampDB int

	// mixrampdelay in seconds
	MixRampDelay int

	// The format emitted by the decoder plugin during playback, format: samplerate:bits:channels. See Global Audio Format for a detailed explanation.
	Audio string

	// Job id
	UpdatingDB string

	// if there is an error, returns message here
	Error string
}

func ParseAttr(attrs mpd.Attrs) *Status {
	status := &Status{}
	for k, v := range attrs {
		switch k {
		case "partition":
			status.Partition = v
		case "volume":
			status.Volume, _ = strconv.Atoi(v)
		case "repeat":
			status.Repeat = v == "1"
		case "random":
			status.Random = v == "1"
		case "single":
			status.Single = ParseOffOnOneshot(v)
		case "consume":
			status.Consume = ParseOffOnOneshot(v)
		case "playlist":
			if i, err := strconv.ParseUint(v, 10, 32); err == nil {
				status.Playlist = uint32(i)
			}
		case "playlistlength":
			if i, err := strconv.ParseInt(v, 10, 32); err == nil {
				status.PlaylistLength = int(i)
			}
		case "state":
			status.State = PlayerState(v)
		case "song":
			status.Song, _ = strconv.Atoi(v)
		case "songid":
			status.SongId, _ = strconv.Atoi(v)
		case "nextsong":
			status.NextSong, _ = strconv.Atoi(v)
		case "nextsongid":
			status.NextSongId, _ = strconv.Atoi(v)
		case "time":
			status.Time, _ = strconv.Atoi(v)
		case "elapsed":
			if f, err := strconv.ParseFloat(v, 32); err == nil {
				status.Elapsed = float32(f)
			}
		case "duration":
			status.Duration, _ = strconv.Atoi(v)
		case "bitrate":
			status.Bitrate, _ = strconv.Atoi(v)
		case "xfade":
			status.CrossFade, _ = strconv.Atoi(v)
		case "mixrampdb":
			status.MixRampDB, _ = strconv.Atoi(v)
		case "mixrampdelay":
			status.MixRampDelay, _ = strconv.Atoi(v)
		case "audio":
			status.Audio = v
		case "updating_db":
			status.UpdatingDB = v
		case "error":

		}
	}
	return status
}
