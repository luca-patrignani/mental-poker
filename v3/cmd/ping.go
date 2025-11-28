package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/luca-patrignani/mental-poker/v3/discovery"
)

type Pinger struct {
	Infos    chan Info
	discover *discovery.Discover
	done     chan struct{}
}

type Info struct {
	Name    string
	Address string
}

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

func (p *Pinger) Close() error {
	return p.discover.Close()
}
