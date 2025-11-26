package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/luca-patrignani/mental-poker/v2/discovery"
)

type Pinger struct {
	discover *discovery.Discover
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
		discover: &discover,
	}
	return &p, nil
}

func (p *Pinger) Start() error {
	return p.discover.Start()
}

func (p *Pinger) PlayersStatus() map[Info]time.Time {
	playersStatus := make(map[Info]time.Time)
	errors := 0
	for {
		select {
		case entry := <-p.discover.Entries:
			info := Info{}
			if err := json.Unmarshal(entry.Info, &info); err != nil {
				fmt.Println(err, errors)
				errors++
				continue
			}
			playersStatus[info] = entry.Time
		default:
			return playersStatus
		}
	}
}

func (p *Pinger) Close() error {
	return p.discover.Close()
}
