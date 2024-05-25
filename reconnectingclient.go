package mmpd

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fhs/gompd/v2/mpd"
	"github.com/go-test/deep"
	"github.com/linkdata/deadlock"
)

var ErrNotConnected = errors.New("not connected")

type ReconnectingClient struct {
	*mpd.Client
	network                     string
	addr                        string
	password                    string
	keepalive                   bool
	pingFunc                    PingFunc
	blocking                    bool
	watchSubsystems             []Subsystem
	connectLock                 deadlock.RWMutex
	idleStateLock               deadlock.Mutex
	activeCommands              int
	isConnected                 atomic.Bool
	closeCh                     chan struct{}
	keepaliveTicker             *time.Ticker
	PlaylistCache               atomic.Pointer[Playlist]
	StatusCache                 atomic.Pointer[Status]
	CurrentSongCache            atomic.Pointer[CurrentSong]
	ConnectedListeners          *ListenerSet[*ConnectedListener]
	DisconnectedListeners       *ListenerSet[*DisconnectedListener]
	SubsystemsChangedListeners  *ListenerSet[*SubsystemsChangedListener]
	StatusChangedListeners      *ListenerSet[*StatusChangedListener]
	PlaylistChangedListeners    *ListenerSet[*PlaylistChangedListener]
	CurrentSongChangedListeners *ListenerSet[*CurrentSongChangedListener]
}

type ClientOption func(*ReconnectingClient)

func WithBlocking() ClientOption {
	return func(client *ReconnectingClient) {
		client.blocking = true
	}
}

func WithPassword(password string) ClientOption {
	return func(client *ReconnectingClient) {
		client.password = password
	}
}

func WithKeepalive(keepalive bool) ClientOption {
	return func(client *ReconnectingClient) {
		client.keepalive = keepalive
	}
}

type PingFunc func(client *ReconnectingClient) error

// WithPingFunc sets an alternative function for keepalive testing.
//
// This can e.g. be used to replace the ping with a status check.
func WithPingFunc(pingFunc PingFunc) ClientOption {
	return func(client *ReconnectingClient) {
		// we expect a non-nil func
		if pingFunc != nil {
			client.pingFunc = pingFunc
		}
	}
}

func WithWatchSubsystems(subsystems ...Subsystem) ClientOption {
	return func(client *ReconnectingClient) {
		client.watchSubsystems = subsystems
	}
}

func NewReconnectingClient(network, addr string, options ...ClientOption) (*ReconnectingClient, error) {
	c := &ReconnectingClient{
		network:                     network,
		addr:                        addr,
		keepalive:                   true,
		pingFunc:                    RefreshCache,
		ConnectedListeners:          NewListenerSet[*ConnectedListener](),
		DisconnectedListeners:       NewListenerSet[*DisconnectedListener](),
		SubsystemsChangedListeners:  NewListenerSet[*SubsystemsChangedListener](),
		StatusChangedListeners:      NewListenerSet[*StatusChangedListener](),
		PlaylistChangedListeners:    NewListenerSet[*PlaylistChangedListener](),
		CurrentSongChangedListeners: NewListenerSet[*CurrentSongChangedListener](),
	}
	for _, option := range options {
		option(c)
	}

	if c.blocking {
		fmt.Printf("mpd: connecting (blocking) to %s %s\n", network, addr)
		if err := c.Connect(); err != nil {
			fmt.Printf("mpd: connect to %s %s failed: %v\n", network, addr, err)
			return nil, err
		} else {
			fmt.Printf("mpd: connect to %s %s succeeded\n", network, addr)
			return c, nil
		}
	} else {
		fmt.Printf("mpd: connecting to %s %s in separate goroutine\n", network, addr)
		go func() {
			if err := c.Connect(); err != nil {
				fmt.Printf("mpd: background connect to %s %s failed: %v; starting reconnect...\n", network, addr, err)
				c.reconnect()
			} else {
				fmt.Printf("mpd: background connect to %s %s succeeded\n", network, addr)
			}
		}()
		return c, nil
	}
}

func (c *ReconnectingClient) Connect() error {
	c.connectLock.Lock()
	defer c.connectLock.Unlock()

	return c.connect()
}

func (c *ReconnectingClient) connect() error {
	if client, err := mpd.DialAuthenticated(c.network, c.addr, c.password); err != nil {
		return err
	} else {
		c.Client = client
		c.closeCh = make(chan struct{})
		c.isConnected.Store(true)
		if c.keepalive {
			c.keepaliveTicker = time.NewTicker(time.Minute)
			go func() {
				for {
					select {
					case <-c.closeCh:
						return
					case <-c.keepaliveTicker.C:
						c.connectLock.RLock()
						if err := c.pingFunc(c); err != nil {
							c.connectLock.RUnlock()
							fmt.Printf("mpd: keepalive ping failed: %v; starting reconnect...\n", err)
							c.isConnected.Store(false)
							c.reconnect()
						} else {
							c.connectLock.RUnlock()
						}
					}
				}
			}()
		}

		fmt.Printf("mpd: notifying connected to %s %s\n", c.network, c.addr)
		// allow the listeners to acquire the connectLock
		go c.ConnectedListeners.Notify(func(l *ConnectedListener) { l.Connected(c) })

		if c.keepalive {
			go c.pingFunc(c)
		}
		return nil
	}
}

func (c *ReconnectingClient) reconnect() {
	// get rid of old client
	c.connectLock.Lock()
	defer c.connectLock.Unlock()

	_ = c.close()

	t0 := time.Now()
connect:
	if err := c.connect(); err != nil {
		fmt.Printf("mpd: reconnect failed: %v\n", err)
		select {
		case <-c.closeCh:
			fmt.Printf("mpd: reconnect aborted due to close\n")
			return
		case <-time.After(time.Second):
			fmt.Printf("mpd: retrying connect due to failed reconnect\n")
			goto connect
		}
	} else {
		fmt.Printf("mpd: reconnect succeeded after %s\n", time.Since(t0).String())
	}
}

func (c *ReconnectingClient) Close() error {
	close(c.closeCh) // outside connectLock

	c.connectLock.Lock()
	defer c.connectLock.Unlock()

	return c.close()
}

func (c *ReconnectingClient) close() error {
	if c.keepaliveTicker != nil {
		c.keepaliveTicker.Stop()
		c.keepaliveTicker = nil
	}

	if c.Client != nil {
		err := c.Client.Close()
		c.Client = nil
		c.isConnected.Store(false)
		fmt.Printf("mpd: notifying disconnected from %s %s\n", c.network, c.addr)
		// allow the listeners to acquire the connectLock
		go c.DisconnectedListeners.Notify(func(l *DisconnectedListener) { l.Disconnected(c) })
		return err
	}

	return nil
}

func (c *ReconnectingClient) IsConnected() bool {
	return c.isConnected.Load()
}

// Do runs a client command.
//
// All commands must be run via Do() so they are
// protected by the connectLock and obey correct idle behavior.
func (c *ReconnectingClient) Do(fn func(client *ReconnectingClient) error) error {
	c.connectLock.RLock()
	defer c.connectLock.RUnlock()

	if c.Client == nil {
		return ErrNotConnected
	} else {
		// TODO
		//if c.watchSubsystems != nil {
		//	c.idleStateLock.Lock()
		//	if c.activeCommands == 0 {
		//		// terminate idle command
		//		if subsystems, err := c.noIdle(); err != nil {
		//			c.idleStateLock.Unlock()
		//			return err
		//		} else {
		//			go c.notifySubsystemsChanged(subsystems)
		//		}
		//	}
		//	c.activeCommands++
		//	c.idleStateLock.Unlock()
		//
		//	defer func() {
		//		c.idleStateLock.Lock()
		//		c.activeCommands--
		//		if c.activeCommands == 0 {
		//			// reestablish idle command
		//			if subsystems, err := c.idle(c.watchSubsystems...); err != nil {
		//				return
		//			}
		//		}
		//		c.idleStateLock.Unlock()
		//	}()
		//}
		return fn(c)
	}
}

func (c *ReconnectingClient) idle(subsystems ...Subsystem) ([]Subsystem, error) {
	changed, err := c.Command("idle %s", mpd.Quoted(strings.Join(StringsForSubsystems(subsystems), " "))).Strings("changed")
	return SubsystemsForStrings(changed), err
}

func (c *ReconnectingClient) noIdle() ([]Subsystem, error) {
	subsystems, err := c.Command("noidle").Strings("changed")
	return SubsystemsForStrings(subsystems), err
}

func (c *ReconnectingClient) ReloadStatus() error {
	return RefreshCache(c)
}

func Ping(client *ReconnectingClient) error {
	return client.Ping()
}

func RefreshCache(client *ReconnectingClient) error {
	// This will get called only once per keep-alive for the mpd client instance,
	// so we use listeners to get it to all interested action instances.
	if attrs, err := client.Status(); err != nil {
		return err
	} else {
		status := ParseStatusAttrs(attrs)
		oldStatus := client.StatusCache.Swap(status)

		if oldStatus == nil || status.Playlist != oldStatus.Playlist {
			fmt.Printf("mpd: playlist id changed to %d\n", status.Playlist)
			if attrsList, err := client.PlaylistInfo(-1, -1); err != nil {
				return err
			} else {
				fmt.Printf("mpd: received new playlist #%d len=%d\n", status.Playlist, len(attrsList))
				newPlaylist := NewPlaylist(attrsList)
				client.PlaylistCache.Store(newPlaylist)

				go client.PlaylistChangedListeners.Notify(func(l *PlaylistChangedListener) {
					l.PlaylistChanged(client, newPlaylist)
				})
			}
		}

		if oldStatus == nil || !status.Equals(oldStatus) {
			fmt.Printf("mpd: status changed (%v)\n", deep.Equal(oldStatus, status))
			go client.StatusChangedListeners.Notify(func(l *StatusChangedListener) {
				l.StatusChanged(client, status)
			})

			currentSong := NewCurrentSong(status, client.PlaylistCache.Load())
			oldCurrentSong := client.CurrentSongCache.Swap(currentSong)
			if oldCurrentSong == nil || !currentSong.Equals(oldCurrentSong) {
				fmt.Printf("mpd: current song changed (%v): %#v\n", deep.Equal(oldCurrentSong, currentSong), currentSong)
				go client.CurrentSongChangedListeners.Notify(func(l *CurrentSongChangedListener) {
					l.CurrentSongChanged(client, currentSong)
				})
			}
		}
		return nil
	}
}
