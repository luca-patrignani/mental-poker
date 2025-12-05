package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/luca-patrignani/mental-poker/v3/discovery"
)

// Pinger wraps a discovery.Discover and forwards discovered Info values on the
// Infos channel. Create a Pinger with NewPinger, call Start to begin discovery,
// and call Close to stop it and release resources.
type Pinger struct {
	Infos    chan Info
	discover *discovery.Discover
	done     chan struct{}
}

// Info is the JSON-serializable payload announced by each node. Pinger expects
// announcements to be JSON-encoded Info objects.
type Info struct {
	Name    string
	Address string
}

// NewPinger returns a configured Pinger that will announce the provided Info
// at the given interval. The returned Pinger is not started; call Start to
// begin network activity.
func NewPinger(info Info, intervalBetweenPings time.Duration) (*Pinger, error) {
	infoJson, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	discover := discovery.Discover{
		Info:                         infoJson,
		Port:                         discoveryPort,
		IntervalBetweenAnnouncements: intervalBetweenPings,
	}
	p := Pinger{
		Infos:    make(chan Info),
		discover: &discover,
		done:     make(chan struct{}),
	}
	return &p, nil
}

// Start begins discovery and starts a goroutine which emits newly-seen peers
// on the Infos channel. The caller should read from Infos until Close is called.
func (p *Pinger) Start() error {
	if err := p.discover.Start(); err != nil {
		return err
	}
	go func() {
		players := map[Info]time.Time{}
		for entry := range p.discover.Entries {
			info := Info{}
			if err := json.Unmarshal(entry.Info, &info); err != nil {
				fmt.Println(err)
				continue
			}
			if _, ok := players[info]; !ok {
				p.Infos <- info
				players[info] = entry.Time
			}
		}
	}()
	return nil
}

// Close stops the underlying discovery instance and releases network resources.
func (p *Pinger) Close() error {
	return p.discover.Close()
}
