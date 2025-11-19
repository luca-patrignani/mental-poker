package network

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"testing"
	"time"
)

func createCertificates(addresses map[int]string) (*x509.CertPool, map[int]tls.Certificate, error) {
	certPool := x509.NewCertPool()
	clientCerts := map[int]tls.Certificate{}
	for i, address := range addresses {
		cert, pem, err := GenerateSelfSignedCert(address)
		if err != nil {
			return nil, nil, err
		}
		clientCerts[i] = cert
		certPool.AppendCertsFromPEM(pem)
	}
	return certPool, clientCerts, nil
}

func useHttps(addresses map[int]string) map[int]string {
	httpsAddresses := map[int]string{}
	for i, addr := range addresses {
		httpsAddresses[i] = "https://" + addr
	}
	return httpsAddresses
}

func TestHttpsBroadcast(t *testing.T) {
	n := 4
	listeners, addresses := CreateListeners(n)
	certPool, clientCerts, err := createCertificates(addresses)
	addresses = useHttps(addresses)
	if err != nil {
		t.Fatal(err)
	}
	root := 3
	fatal := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			peer := NewPeerWithOptions(i, addresses,
				WithTimeout(3*time.Second),
				WithCertificate(clientCerts[i]),
				WithLimitedCAs(certPool),
			)
			peer.Start(listeners[i])
			recv, err := peer.Broadcast([]byte{0, byte(10 * i)}, root)
			if err != nil {
				fatal <- err
				return
			}
			if len(recv) != 2 {
				fatal <- fmt.Errorf("expected length 2, %v received", recv)
				return
			}
			if recv[1] != byte(root*10) {
				fatal <- fmt.Errorf("expected %d, actual %d", recv[1], root*10)
				return
			}
			fatal <- peer.Close()
		}(i)
	}
	for i := 0; i < n; i++ {
		if err := <-fatal; err != nil {
			t.Fatal(err)
		}
	}
}

func TestHttpsAllToAll(t *testing.T) {
	n := 4
	listeners, addresses := CreateListeners(n)
	certPool, clientCerts, err := createCertificates(addresses)
	addresses = useHttps(addresses)
	if err != nil {
		t.Fatal(err)
	}
	fatal := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			peer := NewPeerWithOptions(i, addresses,
				WithTimeout(3*time.Second),
				WithLimitedCAs(certPool),
				WithCertificate(clientCerts[i]),
			)
			peer.Start(listeners[i])
			data := []byte(fmt.Sprint(10 * i))
			recv, err := peer.AllToAll(data)
			if err != nil {
				fatal <- err
				return
			}
			if len(recv) != n {
				fatal <- fmt.Errorf("expected length %d, %d received", n, len(recv))
				return
			}
			for j := 0; j < n; j++ {
				if string(recv[j]) != fmt.Sprint(10*j) {
					fatal <- fmt.Errorf("expected %d, actual %d", 10*j, recv[j])
					return
				}
			}
			fatal <- peer.Close()
		}(i)
	}
	for i := 0; i < n; i++ {
		if err := <-fatal; err != nil {
			t.Fatal(err)
		}
	}
}

func TestPeerOptionsHttp(t *testing.T) {
	n := 4
	listeners, addresses := CreateListeners(n)
	fatal := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			peer := NewPeerWithOptions(i, addresses,
				WithTimeout(3*time.Second),
			)
			peer.Start(listeners[i])
			data := []byte(fmt.Sprint(10 * i))
			recv, err := peer.AllToAll(data)
			if err != nil {
				fatal <- err
				return
			}
			if len(recv) != n {
				fatal <- fmt.Errorf("expected length %d, %d received", n, len(recv))
				return
			}
			for j := 0; j < n; j++ {
				if string(recv[j]) != fmt.Sprint(10*j) {
					fatal <- fmt.Errorf("expected %d, actual %d", 10*j, recv[j])
					return
				}
			}
			fatal <- peer.Close()
		}(i)
	}
	for i := 0; i < n; i++ {
		if err := <-fatal; err != nil {
			t.Fatal(err)
		}
	}
}
