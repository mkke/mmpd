package mmpd

type PlayerState string

const (
	Play  PlayerState = "play"
	Stop  PlayerState = "stop"
	Pause PlayerState = "pause"
)
