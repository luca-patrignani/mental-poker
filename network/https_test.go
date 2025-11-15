package network

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
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

const (
	certFile1   = "server1.crt"
	keyFile1    = "server1.key"	
	certFile2   = "server2.crt"
	keyFile2    = "server2.key"
	serverAddr1 = "127.0.0.1:8443"
	serverAddr2 = "127.0.0.1:8444"
)

func generateCertKey(certPath, keyPath string) error {

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("error generating RSA key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("error generating serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Go Localhost Test"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(1, 0, 0),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:    []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("error creating certificate: %w", err)
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("unable to open %s for writing: %w", certPath, err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	certOut.Close()

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to open %s for writing: %w", keyPath, err)
	}
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	keyOut.Close()

	log.Printf("Generated certificate (%s) and key (%s) successfully.\n", certPath, keyPath)
	return nil
}

func startServer(addr, certPath, keyPath string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from HTTPS server! You are connected to %s", r.Host)
	})

	log.Printf("HTTPS server listening on: https://%s\n", addr)

	err := http.ListenAndServeTLS(addr, certPath, keyPath, mux)
	if err != nil {
		log.Fatalf("HTTPS server error: %v", err)
	}
}

func startClient(addr1, addr2, certPath1, certPath2 string) {

	caCertPool := x509.NewCertPool()
	caCert1, err := os.ReadFile(certPath1)
	if err != nil {
		log.Fatalf("Error reading CA certificate: %v", err)
	}
	caCertPool.AppendCertsFromPEM(caCert1)

	caCert2, err := os.ReadFile(certPath2)
	if err != nil {
		log.Fatalf("Error reading CA certificate: %v", err)
	}
	caCertPool.AppendCertsFromPEM(caCert2)

	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: 5 * time.Second,
	}

	for _, addr := range []string{addr1, addr2} {
		url := fmt.Sprintf("https://%s", addr)
		log.Printf("Client: Attempting to connect to %s\n", url)
	
		resp, err := client.Get(url)
		if err != nil {
			log.Fatalf("HTTPS client request error: %v", err)
		}
		defer resp.Body.Close()
	
		_, err = io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Error reading response: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			log.Fatalf("Unexpected status code: %d", resp.StatusCode)
		}
	}
}

func TestMain(t *testing.T) {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	os.Remove(certFile1)
	os.Remove(keyFile1)
	os.Remove(certFile2)
	os.Remove(keyFile2)

	if err := generateCertKey(certFile1, keyFile1); err != nil {
		log.Fatal(err)
	}

	go startServer(serverAddr1, certFile1, keyFile1)

	if err := generateCertKey(certFile2, keyFile2); err != nil {
		log.Fatal(err)
	}

	go startServer(serverAddr2, certFile2, keyFile2)

	time.Sleep(500 * time.Millisecond)

	startClient(serverAddr1, serverAddr2, certFile1, certFile2)
}
