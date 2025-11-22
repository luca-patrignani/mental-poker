package discovery

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type Discover struct {
	Entries chan Entry
	startPort uint16
	endPort   uint16
}

type Entry struct {
	Info string
}

func New(info string) (*Discover, error) {
	return NewWithPortRange(info, 9000, 9010, 2)
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
		if err := server.Serve(l); err != nil {
			panic(err)
		}
	}()
	discover := &Discover{
		Entries:   make(chan Entry),
		startPort: startPort,
		endPort:   endPort,
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
