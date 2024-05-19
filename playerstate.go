package mmpd

// PlayerState represents the current player state.
type PlayerState string

const (
	Play  PlayerState = "play"
	Stop  PlayerState = "stop"
	Pause PlayerState = "pause"
)
