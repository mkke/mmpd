package mmpd

import (
	"fmt"

	"github.com/linkdata/deadlock"
)

type NetAddr struct {
	network string
	address string
}

type ClientCacheEntry struct {
	NetAddr NetAddr
	Options []ClientOption
	*ReconnectingClient
}

type ClientCache struct {
	clients    map[NetAddr]*ClientCacheEntry
	clientLock deadlock.RWMutex
}

func NewClientCache() *ClientCache {
	return &ClientCache{
		clients: make(map[NetAddr]*ClientCacheEntry),
	}
}

func (cc *ClientCache) GetOrCreate(network, addr string, options ...ClientOption) (*ClientCacheEntry, error) {
	netAddr := NetAddr{network: network, address: addr}

	cc.clientLock.RLock()
	if entry, ok := cc.clients[netAddr]; ok {
		cc.clientLock.RUnlock()
		fmt.Printf("mpd: using cached client for target %s %s\n", netAddr.network, netAddr.address)
		return entry, nil
	}
	cc.clientLock.RUnlock()

	cc.clientLock.Lock()
	defer cc.clientLock.Unlock()
	if entry, ok := cc.clients[netAddr]; ok {
		fmt.Printf("mpd: using newly cached client for target %s %s\n", netAddr.network, netAddr.address)
		return entry, nil
	}

	fmt.Printf("mpd: creating new client for target %s %s\n", netAddr.network, netAddr.address)
	client, err := NewReconnectingClient(network, addr, options...)
	if err != nil {
		return nil, err
	}

	cce := &ClientCacheEntry{
		NetAddr:            netAddr,
		Options:            options,
		ReconnectingClient: client,
	}
	cc.clients[netAddr] = cce

	fmt.Printf("mpd: returning new client for target %s %s\n", netAddr.network, netAddr.address)
	return cce, nil
}

func (cc *ClientCache) Shutdown() {
	cc.clientLock.Lock()
	defer cc.clientLock.Unlock()

	for _, cce := range cc.clients {
		_ = cce.Client.Close()
	}

	clear(cc.clients)
}
