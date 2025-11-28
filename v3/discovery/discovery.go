package discovery

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"slices"
	"time"
)

const multicastIpAddress = "239.0.0.1"

type Discover struct {
	Info                         []byte
	Port                         uint16
	IntervalBetweenAnnouncements time.Duration
	Entries                      chan Entry
	conn                         *net.UDPConn
	sendConn                     *net.UDPConn
	key                          []byte
}

type Entry struct {
	Info []byte
	Time time.Time
}

func (d *Discover) Start() error {
	d.Entries = make(chan Entry, 10)
	d.key = []byte(fmt.Sprintf("%08x", rand.Uint32()))
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", multicastIpAddress, d.Port))
	if err != nil {
		return err
	}
	ifi, err := net.InterfaceByName("wlo1")
	if err != nil {
		return err
	}
	d.conn, err = net.ListenMulticastUDP("udp4", ifi, addr)
	if err != nil {
		return err
	}

	d.sendConn, err = net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	d.startListener()
	d.startDialer()
	return nil
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
			message := buffer[:n]
			if slices.Compare(message[:8], d.key) == 0 {
				continue
			}
			d.Entries <- Entry{
				Info: message[8:],
				Time: time.Now(),
			}
			time.Sleep(d.IntervalBetweenAnnouncements)
		}
	}()
}

func (d Discover) startDialer() {
	go func() {
		for {
			if _, err := d.sendConn.Write(append([]byte(d.key), d.Info...)); err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				panic(err)
			}
			time.Sleep(d.IntervalBetweenAnnouncements)
		}
	}()
}
