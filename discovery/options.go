package discovery

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

type Discover struct {
	Entries   chan Entry
	port      uint16
	startPort uint16
	endPort   uint16
	server    *http.Server
	ipAddress string
	attempts  uint
}

type option func(Discover) Discover

func NewWithOptions(info string, opts ...option) (*Discover, error) {
	d := Discover{
		Entries:   make(chan Entry),
		startPort: 9000,
		endPort:   9010,
		attempts: 1,
	}
	for _, opt := range opts {
		d = opt(d)
	}

	var l net.Listener
	var err error
	var port uint16
	for port = d.startPort; port <= d.endPort; port++ {
		l, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			d.port = port
			break
		}
	}
	if err != nil {
		return nil, err
	}
	d.server = &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", port),
		Handler: handler{info: info},
	}
	go func() {
		if err := d.server.Serve(l); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	go func() {
		for range d.attempts {
			d.search()
			time.Sleep(time.Second)
		}
	}()
	return &d, nil
}

func WithPortRange(startPort, endPort uint16) option {
	return func(d Discover) Discover {
		d.startPort = startPort
		d.endPort = endPort
		return d
	}
}

func WithPort(port uint16) option {
	return WithPortRange(port, port)
}

func WithAttempts(attempts uint) option {
	return func(d Discover) Discover {
		d.attempts = attempts
		return d
	}
}