package network

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"testing"
	"time"
)

func generateSelfSignedCert() (tls.Certificate, []byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Mental Poker"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:              []string{"localhost"},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	certPEMBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})
	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
	}
	return cert, certPEMBytes, nil
}

func createHttpsListeners(n int) (map[int]net.Listener, map[int]string, *x509.CertPool, error) {
	listeners, addresses := CreateListeners(n)
	certPool := x509.NewCertPool()
	for i := 0; i < n; i++ {
		cert, pem, err := generateSelfSignedCert()
		if err != nil {
			return nil, nil, nil, err
		}
		certPool.AppendCertsFromPEM(pem)
		listeners[i] = tls.NewListener(listeners[i], &tls.Config{
			Certificates: []tls.Certificate{cert},
		})
		addresses[i] = "https://" + addresses[i]
	}
	return listeners, addresses, certPool, nil
}

func TestHttpsBroadcast(t *testing.T) {
	n := 4
	listeners, addresses, certPool, err := createHttpsListeners(n)
	if err != nil {
		t.Fatal(err)
	}
	root := 3
	fatal := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			peer := NewPeerHttps(i, addresses, listeners[i], 50*time.Second, certPool)
			defer func() {
				fatal <- peer.Close()
			}()
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
	listeners, addresses, certPool, err := createHttpsListeners(n)
	if err != nil {
		t.Fatal(err)
	}
	fatal := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			peer := NewPeerHttps(i, addresses, listeners[i], 50*time.Second, certPool)
			defer func() {
				fatal <- peer.Close()
			}()
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
		}(i)
	}
	for i := 0; i < n; i++ {
		if err := <-fatal; err != nil {
			t.Fatal(err)
		}
	}
}
