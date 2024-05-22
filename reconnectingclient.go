package mmpd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fhs/gompd/v2/mpd"
	"github.com/linkdata/deadlock"
)

var ErrNotConnected = errors.New("not connected")

type ReconnectingClient struct {
	*mpd.Client
	network                    string
	addr                       string
	password                   string
	keepalive                  bool
	pingFunc                   PingFunc
	blocking                   bool
	watchSubsystems            []Subsystem
	connectLock                deadlock.RWMutex
	idleStateLock              deadlock.Mutex
	activeCommands             int
	closeCh                    chan struct{}
	keepaliveTicker            *time.Ticker
	Playlist                   []PlaylistEntry
	ConnectedListeners         *ListenerSet[*ConnectedListener]
	DisconnectedListeners      *ListenerSet[*DisconnectedListener]
	SubsystemsChangedListeners *ListenerSet[*SubsystemsChangedListener]
	StatusChangedListeners     *ListenerSet[*StatusChangedListener]
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
		network:   network,
		addr:      addr,
		keepalive: true,
		pingFunc: func(client *ReconnectingClient) error {
			return client.Ping()
		},
		ConnectedListeners:         NewListenerSet[*ConnectedListener](),
		DisconnectedListeners:      NewListenerSet[*DisconnectedListener](),
		SubsystemsChangedListeners: NewListenerSet[*SubsystemsChangedListener](),
		StatusChangedListeners:     NewListenerSet[*StatusChangedListener](),
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
		fmt.Printf("mpd: notifying disconnected from %s %s\n", c.network, c.addr)
		// allow the listeners to acquire the connectLock
		go c.DisconnectedListeners.Notify(func(l *DisconnectedListener) { l.Disconnected(c) })
		return err
	}

	return nil
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
