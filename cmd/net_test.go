package main

import (
	"net"
	"testing"

)

func TestGuessIpAddress24(t *testing.T) {
	addr := net.IP{192, 168, 0, 1}
	actual, err := guessIpAddress(addr, "42")
	if err != nil {
		t.Fatal(err)
	}
	expected := net.IP{192, 168, 0, 42}
	if !actual.Equal(expected) {
		t.Fatalf("expected %v, actual %v", expected, actual)
	}
}

func TestGuessIpAddress16(t *testing.T) {
	addr := net.IP{192, 168, 0, 1}
	actual, err := guessIpAddress(addr, "15.42")
	if err != nil {
		t.Fatal(err)
	}
	expected := net.IP{192, 168, 15, 42}
	if !actual.Equal(expected) {
		t.Fatalf("expected %v, actual %v", expected, actual)
	}
}

func TestGuessIpAddress0(t *testing.T) {
	addr := net.IP{192, 168, 0, 1}
	actual, err := guessIpAddress(addr, "10.100.15.42")
	if err != nil {
		t.Fatal(err)
	}
	expected := net.IP{10, 100, 15, 42}
	if !actual.Equal(expected) {
		t.Fatalf("expected %v, actual %v", expected, actual)
	}
}

func TestGuessIpAddress32(t *testing.T) {
	addr := net.IP{192, 168, 0, 1}
	actual, err := guessIpAddress(addr, "")
	if err != nil {
		t.Fatal(err)
	}
	if !actual.Equal(addr) {
		t.Fatalf("expected %v, actual %v", addr, actual)
	}
}


func TestSubnetOfListener(t *testing.T) {
	l, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 12345,
	})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()

	ipnet, err := subnetOfListener(l)
	if err != nil {
		t.Fatalf("SubnetOfListener error: %v", err)
	}
	t.Logf("listener local addr: %v, subnet: %s", l.Addr(), ipnet.String())

	if !ipnet.Contains(net.ParseIP("127.0.0.1")) {
		t.Fatalf("expected subnet %s to contain 127.0.0.1", ipnet.String())
	}
}
