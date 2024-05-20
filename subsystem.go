package mmpd

type Subsystem string

func SubsystemsForStrings(strs []string) (subsystems []Subsystem) {
	for _, s := range strs {
		subsystems = append(subsystems, Subsystem(s))
	}
	return subsystems
}

func StringsForSubsystems(subsystems []Subsystem) (strings []string) {
	for _, s := range subsystems {
		strings = append(strings, string(s))
	}
	return strings
}

const (
	// the song database has been modified after update.
	SubsystemDatabase Subsystem = "database"

	// a database update has started or finished. If the database was modified during the update, the database event is also emitted.
	SubsystemUpdate Subsystem = "update"

	// a stored playlist has been modified, renamed, created or deleted
	SubsystemStoredPlaylist Subsystem = "stored_playlist"

	// the queue (i.e. the current playlist) has been modified
	SubsystemPlaylist Subsystem = "playlist"

	// the player has been started, stopped or seeked or tags of the currently playing song have changed (e.g. received from stream)
	SubsystemPlayer Subsystem = "player"

	// the volume has been changed
	SubsystemMixer Subsystem = "mixer"

	// an audio output has been added, removed or modified (e.g. renamed, enabled or disabled)
	SubsystemOutput Subsystem = "output"

	// options like repeat, random, crossfade, replay gain
	SubsystemOptions Subsystem = "options"

	// a partition was added, removed or changed
	SubsystemPartition Subsystem = "partition"

	// the sticker database has been modified.
	SubsystemSticker Subsystem = "sticker"

	// a client has subscribed or unsubscribed to a channel
	SubsystemSubscription Subsystem = "subscription"

	// a message was received on a channel this client is subscribed to; this event is only emitted when the client’s message queue is empty
	SubsystemMessage Subsystem = "message"

	// a neighbor was found or lost
	SubsystemNeighbour Subsystem = "neighbour"

	// the mount list has changed
	SubsystemMount Subsystem = "mount"
)
