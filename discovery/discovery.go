package discovery

import (
	"context"
	"fmt"
	"io"
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

type Entry struct {
	Info string
}

type option func(Discover) Discover

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

func New(info string, opts ...option) (*Discover, error) {
	d := Discover{
		Entries:   make(chan Entry),
		startPort: 9000,
		endPort:   9000,
		attempts:  1,
	}
	for _, opt := range opts {
		d = opt(d)
	}

	var l net.Listener
	var err error
	for port := d.startPort; port <= d.endPort; port++ {
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
		Addr:    fmt.Sprintf("localhost:%d", d.port),
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

func (d *Discover) Close() error {
	return d.server.Shutdown(context.Background())
}

func (d *Discover) search() {
	for port := d.startPort; port <= d.endPort; port++ {
		if port == d.port {
			continue
		}
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d", port))
		if err != nil {
			continue
		}
		buf, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		d.Entries <- Entry{
			Info: string(buf),
		}
	}
}

type handler struct {
	info string
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte(h.info)); err != nil {
		panic(err)
	}
}
