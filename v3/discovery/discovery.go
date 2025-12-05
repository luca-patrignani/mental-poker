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

// Discover represents a discovery instance that announces and listens for service information.
// Before calling Start, configure Info (the payload to announce), Port (UDP port), and
// IntervalBetweenAnnouncements (frequency of announcements). After Start succeeds,
// discovered entries are received on the Entries channel.
type Discover struct {
	Info                         []byte
	Port                         uint16
	IntervalBetweenAnnouncements time.Duration
	Entries                      chan Entry
	conn                         *net.UDPConn
	sendConn                     *net.UDPConn
	key                          []byte
}

// Entry represents a single discovery announcement received from a peer.
// Info contains the service information payload and Time is when it was received.
type Entry struct {
	Info []byte
	Time time.Time
}

// Start initializes the discovery mechanism: joins the multicast group, creates
// network connections, and starts background goroutines for listening and announcing.
// Returns an error if network setup fails. On success, the Entries channel will
// receive discovered entries from other peers.
func (d *Discover) Start() error {
	d.Entries = make(chan Entry, 10)
	d.key = []byte(fmt.Sprintf("%08x", rand.Uint32()))
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", multicastIpAddress, d.Port))
	if err != nil {
		return err
	}
	d.conn, err = net.ListenMulticastUDP("udp", nil, addr)
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

// Close stops the discovery mechanism and closes the underlying UDP connections.
// Returns a combined error if either connection closure fails.
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
