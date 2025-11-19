package network

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"time"
)

type peerOption func(Peer) Peer

func NewPeerWithOptions(rank int, addresses map[int]string, opts ...peerOption) Peer {
	handler := &broadcastHandler{
		contentChannel: make(chan []byte),
		errChannel:     make(chan error),
	}
	tlsConfig := &tls.Config{}
	p := Peer{
		Rank:      rank,
		Addresses: copyMap(addresses),
		clock:     0,
		server:    &http.Server{Addr: addresses[rank], Handler: handler},
		handler:   handler,
		tlsConfig: tlsConfig,
		client: http.Client{},
	}
	for _, opt := range opts {
		p = opt(p)
	}
	return p
}

func (p Peer) Start(l net.Listener) {
	if p.tlsConfig.Certificates != nil {
		l = tls.NewListener(l, p.tlsConfig)
	}
	go func() {
		err := p.server.Serve(l)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
}

func WithTimeout(timeout time.Duration) peerOption {
	return func(p Peer) Peer {
		p.timeout = timeout
		return p
	}
}

func WithCertificate(cert tls.Certificate) peerOption {
	return func(p Peer) Peer {
		if p.client.Transport == nil {
			p.client.Transport = &http.Transport{TLSClientConfig: p.tlsConfig}
		}
		p.tlsConfig.Certificates = append(p.tlsConfig.Certificates, cert)
		return p
	}
}

func WithLimitedCAs(certPool *x509.CertPool) peerOption {
	return func(p Peer) Peer {
		if p.client.Transport == nil {
			p.client.Transport = &http.Transport{TLSClientConfig: p.tlsConfig}
		}
		p.tlsConfig.RootCAs = certPool
		p.tlsConfig.ClientCAs = certPool
		p.tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		return p
	}
}
