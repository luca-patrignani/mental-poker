package discovery

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

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

func NewWithPortRange(info string, startPort, endPort uint16, attempts uint) (*Discover, error) {
	return NewWithOptions(info, 
		WithPortRange(startPort, endPort),
		WithAttempts(attempts),
	)
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
