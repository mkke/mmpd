package mmpd

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/fhs/gompd/v2/mpd"
)

type Playlist struct {
	Entries []*PlaylistEntry
}

func NewPlaylist(attrsList []mpd.Attrs) *Playlist {
	entries := make([]*PlaylistEntry, len(attrsList))
	for idx, attrs := range attrsList {
		entries[idx] = ParsePlaylistEntryAttrs(attrs)
	}
	return &Playlist{Entries: entries}
}

// PlaylistEntry represents song attributes of a playlist entry.
type PlaylistEntry struct {
	// the artist name. Its meaning is not well-defined; see “composer” and “performer” for more specific tags.
	Artist string

	// same as artist, but for sorting. This usually omits prefixes such as “The”.
	ArtistSort string

	// the album name.
	Album string

	// same as album, but for sorting.
	AlbumSort string

	// on multi-artist albums, this is the artist name which shall be used for the whole album. The exact meaning of this tag is not well-defined.
	AlbumArtist string

	// same as albumartist, but for sorting.
	AlbumArtistSort string

	// the song title.
	Title string

	// same as title, but for sorting.
	TitleSort string

	// the decimal track number within the album.
	Track int

	// a name for this song. This is not the song title. The exact meaning of this tag is not well-defined. It is often used by badly configured internet radio stations with broken tags to squeeze both the artist name and the song title in one tag.
	Name string

	// the music genre.
	Genre string

	// the mood of the audio with a few keywords.
	Mood string

	// the song’s release date. This is usually a 4-digit year.
	Date string

	// the song’s original release date.
	OriginalDate string

	// the artist who composed the song.
	Composer string

	// same as composer, but for sorting.
	ComposerSort string

	// the artist who performed the song.
	Performer string

	// the conductor who conducted the song.
	Conductor string

	// a work is a distinct intellectual or artistic creation, which can be expressed in the form of one or more audio recordings
	Work string

	// the ensemble performing this song, e.g. “Wiener Philharmoniker”.
	Ensemble string

	// name of the movement, e.g. “Andante con moto”.
	Movement string

	// movement number, e.g. “2” or “II”.
	MovementNumber string

	// location of the recording, e.g. “Royal Albert Hall”.
	Location string

	// "used if the sound belongs to a larger category of sounds/music" (from the IDv2.4.0 TIT1 description).
	Grouping string

	// a human-readable comment about this song. The exact meaning of this tag is not well-defined.
	Comment string

	// the decimal disc number in a multi-disc album.
	Disc int

	// the name of the label or publisher.
	Label string

	// the artist id in the MusicBrainz database.
	MusicbrainzArtistId string

	// the album id in the MusicBrainz database.
	MusicbrainzAlbumId string

	// the album artist id in the MusicBrainz database.
	MusicbrainzAlbumArtistId string

	// the track id in the MusicBrainz database.
	MusicbrainzTrackId string

	// the release group id in the MusicBrainz database.
	MusicbrainzReleaseGroupId string

	// the release track id in the MusicBrainz database.
	MusicbrainzReleaseTrackId string

	// the work id in the MusicBrainz database.
	MusicbrainzWorkId string

	// the duration of the song in seconds; may contain a fractional part.
	Duration float32

	// like duration, but as integer value. This is deprecated and is only here for compatibility with older clients. Do not use.
	Time int

	// if this is a queue item referring only to a portion of the song file, then this attribute contains the time range in the form START-END or START- (open ended); both START and END are time stamps within the song in seconds (may contain a fractional part). Example: 60-120 plays only the second minute; “180 skips the first three minutes.
	Range string

	// the audio format of the song (or an approximation to a format supported by MPD and the decoder plugin being used). When playing this file, the audio value in the status response should be the same.
	Format string

	// the time stamp of the last modification of the underlying file in ISO 8601 format. Example: “2008-09-28T20:04:57Z”
	LastModified string

	// the time stamp when the file was added in ISO 8601. A negative value means that this is unknown/unavailable. Example: “2023-11-25T13:25:07Z”
	Added string
}

func (pe *PlaylistEntry) Equals(other *PlaylistEntry) bool {
	if pe == nil && other == nil {
		return true
	} else if pe != nil && other != nil {
		return reflect.DeepEqual(pe, other)
	} else {
		return false
	}
}

func ParsePlaylistEntryAttrs(attrs mpd.Attrs) *PlaylistEntry {
	entry := &PlaylistEntry{}
	for k, v := range attrs {
		switch strings.ToLower(k) {
		case "artist":
			entry.Artist = v
		case "artistsort":
			entry.ArtistSort = v
		case "album":
			entry.Album = v
		case "albumsort":
			entry.AlbumSort = v
		case "albumartist":
			entry.AlbumArtist = v
		case "albumartistsort":
			entry.AlbumArtistSort = v
		case "title":
			entry.Title = v
		case "titlesort":
			entry.TitleSort = v
		case "track":
			entry.Track, _ = strconv.Atoi(v)
		case "name":
			entry.Name = v
		case "genre":
			entry.Genre = v
		case "mood":
			entry.Mood = v
		case "date":
			entry.Date = v
		case "originaldate":
			entry.OriginalDate = v
		case "composer":
			entry.Composer = v
		case "composersort":
			entry.ComposerSort = v
		case "performer":
			entry.Performer = v
		case "conductor":
			entry.Conductor = v
		case "work":
			entry.Work = v
		case "ensemble":
			entry.Ensemble = v
		case "movement":
			entry.Movement = v
		case "movementnumber":
			entry.MovementNumber = v
		case "location":
			entry.Location = v
		case "grouping":
			entry.Grouping = v
		case "comment":
			entry.Comment = v
		case "disc":
			entry.Disc, _ = strconv.Atoi(v)
		case "label":
			entry.Label = v
		case "musicbrainz_artistid":
			entry.MusicbrainzArtistId = v
		case "musicbrainz_albumid":
			entry.MusicbrainzAlbumId = v
		case "musicbrainz_albumartistid":
			entry.MusicbrainzAlbumArtistId = v
		case "musicbrainz_trackid":
			entry.MusicbrainzTrackId = v
		case "musicbrainz_releasegroupid":
			entry.MusicbrainzReleaseGroupId = v
		case "musicbrainz_releasetrackid":
			entry.MusicbrainzReleaseTrackId = v
		case "musicbrainz_workid":
			entry.MusicbrainzWorkId = v
		case "duration":
			if f, err := strconv.ParseFloat(v, 32); err == nil {
				entry.Duration = float32(f)
			}
		case "time":
			entry.Time, _ = strconv.Atoi(v)
		case "range":
			entry.Range = v
		case "format":
			entry.Format = v
		case "lastmodified":
			entry.LastModified = v
		case "added":
			entry.Added = v

		}
	}
	return entry
}
