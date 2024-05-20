package mmpd

import (
	"errors"
	"fmt"
	"time"

	"github.com/fhs/gompd/v2/mpd"
	"github.com/linkdata/deadlock"
)

var ErrNotConnected = errors.New("not connected")

type ReconnectingClient struct {
	*mpd.Client
	network               string
	addr                  string
	password              string
	keepalive             bool
	blocking              bool
	lock                  deadlock.RWMutex
	closeCh               chan struct{}
	keepaliveTicker       *time.Ticker
	Playlist              []PlaylistEntry
	connectedListeners    map[*ConnectedListener]struct{}
	disconnectedListeners map[*DisconnectedListener]struct{}
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

func NewReconnectingClient(network, addr string, options ...ClientOption) (*ReconnectingClient, error) {
	c := &ReconnectingClient{
		network:               network,
		addr:                  addr,
		keepalive:             true,
		connectedListeners:    map[*ConnectedListener]struct{}{},
		disconnectedListeners: map[*DisconnectedListener]struct{}{},
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
	c.lock.Lock()
	defer c.lock.Unlock()

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
						c.lock.RLock()
						if err := c.Client.Ping(); err != nil {
							c.lock.RUnlock()
							fmt.Printf("mpd: keepalive ping failed: %v; starting reconnect...\n", err)
							c.reconnect()
						} else {
							c.lock.RUnlock()
						}
					}
				}
			}()
		}
		fmt.Printf("mpd: notifying connected to %s %s\n", c.network, c.addr)
		// allow the listeners to acquire the lock
		go func() {
			for l := range c.connectedListeners {
				l.fn(c)
			}
		}()
		return nil
	}
}

func (c *ReconnectingClient) reconnect() {
	// get rid of old client
	c.lock.Lock()
	defer c.lock.Unlock()

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
	close(c.closeCh) // outside lock

	c.lock.Lock()
	defer c.lock.Unlock()

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
		// allow the listeners to acquire the lock
		go func() {
			for l := range c.disconnectedListeners {
				l.fn(c)
			}
		}()
		return err
	}

	return nil
}

func (c *ReconnectingClient) Do(fn func(client *ReconnectingClient) error) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.Client == nil {
		return ErrNotConnected
	} else {
		return fn(c)
	}
}

type ConnectedListener struct {
	fn func(client *ReconnectingClient)
}

func NewConnectedListener(fn func(client *ReconnectingClient)) *ConnectedListener {
	return &ConnectedListener{
		fn: fn,
	}
}

func (c *ReconnectingClient) AddConnectedListener(l *ConnectedListener) {
	c.connectedListeners[l] = struct{}{}
}

func (c *ReconnectingClient) RemoveConnectedListener(l *ConnectedListener) {
	delete(c.connectedListeners, l)
}

type DisconnectedListener struct {
	fn func(client *ReconnectingClient)
}

func NewDisconnectedListener(fn func(client *ReconnectingClient)) *DisconnectedListener {
	return &DisconnectedListener{
		fn: fn,
	}
}

func (c *ReconnectingClient) AddOnDisconnectedListener(l *DisconnectedListener) {
	c.disconnectedListeners[l] = struct{}{}
}

func (c *ReconnectingClient) RemoveDisconnectedListener(l *DisconnectedListener) {
	delete(c.disconnectedListeners, l)
}
