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
	Entries chan Entry
	port	 uint16
	startPort uint16
	endPort   uint16
	server  *http.Server
}

type Entry struct {
	Info string
}

func New(info string, port uint16) (*Discover, error) {
	return NewWithPortRange(info, port, port, 2)
}

type handler struct {
	info string
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte(h.info)); err != nil {
		panic(err)
	}
}

func NewWithPortRange(info string, startPort, endPort uint16, attempts int) (*Discover, error) {
	var l net.Listener
	var err error
	var port uint16
	for port = startPort; port <= endPort; port++ {
		l, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	server := http.Server{
		Addr: fmt.Sprintf("localhost:%d", port),
		Handler: handler{info: info},
	}
	go func() {
		if err := server.Serve(l); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	discover := &Discover{
		Entries:   make(chan Entry),
		port:      port,
		startPort: startPort,
		endPort:   endPort,
		server: &server,
	}
	go func() {
		for range attempts {
			discover.search()
			time.Sleep(time.Second)
		}
	}()
	return discover, nil
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

func (d *Discover) Close() error {
	return d.server.Shutdown(context.Background())
}
