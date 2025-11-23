package discovery

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"time"
)

const multicastIpAddress = "239.0.0.1"

type Discover struct {
	Entries                      chan Entry
	Info                         string
	port                         uint16
	conn                         *net.UDPConn
	sendConn                     *net.UDPConn
	intervalBetweenAnnouncements time.Duration
	key                          string
}

type Entry struct {
	Info string
	Time time.Time
}

type option func(Discover) Discover

func WithIntervalBetweenAnnouncements(i time.Duration) option {
	return func(d Discover) Discover {
		d.intervalBetweenAnnouncements = i
		return d
	}
}

func New(info string, port uint16, opts ...option) (*Discover, error) {
	d := Discover{
		Entries:                      make(chan Entry, 100),
		Info:                         info,
		port:                         port,
		intervalBetweenAnnouncements: time.Second,
		key:                          fmt.Sprintf("%08x", rand.Uint32()),
	}
	for _, opt := range opts {
		d = opt(d)
	}
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", multicastIpAddress, d.port))
	if err != nil {
		panic(err)
	}
	d.conn, err = net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}

	sendAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", multicastIpAddress, d.port))
	if err != nil {
		return nil, err
	}
	d.sendConn, err = net.DialUDP("udp", nil, sendAddr)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (d *Discover) Start() {
	d.startListener()
	d.startDialer()
}

func (d *Discover) Close() error {
	err1 := d.conn.Close()
	err2 := d.sendConn.Close()
	return errors.Join(err1, err2)
}

func (d Discover) startListener() {
	go func() {
		for {
			buffer := make([]byte, 1024)
			n, _, err := d.conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					panic(err)
				}
				if errors.Is(err, net.ErrClosed) {
					return
				}
				panic(err)
			}
			message := string(buffer[:n])
			if message[:8] == d.key {
				continue
			}
			d.Entries <- Entry{
				Info: message[8:],
				Time: time.Now(),
			}
		}
	}()
}

func (d Discover) startDialer() {
	go func() {
		for {
			if _, err := d.sendConn.Write(append([]byte(d.key), []byte(d.Info)...)); err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				panic(err)
			}
			time.Sleep(d.intervalBetweenAnnouncements)
		}
	}()
}
