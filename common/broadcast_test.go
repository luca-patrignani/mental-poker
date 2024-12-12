package common

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"
)

func createAddresses(n int) []net.TCPAddr {
	addresses := []net.TCPAddr{}
	for i := 0; i < n; i++ {
		addresses = append(addresses, net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 50000 + i,
		})
		if addresses[i].IP == nil {
			panic(addresses[i].IP)
		}
	}
	return addresses
}

func TestAllToAll(t *testing.T) {
	n := 10
	addresses := createAddresses(n)
	fatal := make(chan error, 3*n)
	for i := 0; i < n; i++ {
		go func() {
			p := Player{
				Rank:      i,
				Addresses: addresses,
			}
			actual, err := p.AllToAll([]byte(strconv.Itoa(i)))
			if err != nil {
				fatal <- err
				return
			}
			if len(actual) != n {
				fatal <- fmt.Errorf("from player %d: expected list of length %d, %v given", i, n, actual)
				return
			}
			for j := 0; j < n; j++ {
				if strconv.Itoa(j) != string(actual[j]) {
					fatal <- fmt.Errorf("from player %d: expected %d, actual %v", i, j, actual[j])
					return
				}
			}
			fatal <- nil
		}()
	}
	for i := 0; i < n; i++ {
		err := <-fatal
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestBroadcast(t *testing.T) {
	n := 10
	addresses := createAddresses(n)
	root := 3
	fatal := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			p := Player{
				Rank:      i,
				Addresses: addresses,
			}
			time.Sleep(time.Millisecond * 100 * time.Duration(p.Rank))
			recv, err := p.Broadcast([]byte{0, byte(10 * i)}, root)
			time.Sleep(time.Millisecond * 100 * time.Duration(p.Rank))
			if err != nil {
				fatal <- err
				return
			}
			if len(recv) != 2 {
				fatal <- fmt.Errorf("expected length 2, %v received", recv)
			}
			if recv[1] != byte(root*10) {
				fatal <- fmt.Errorf("expected %d, actual %d", recv[1], root*10)
				return
			}
			fatal <- nil
		}(i)
	}
	for i := 0; i < n; i++ {
		err := <-fatal
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestBroadcastTwoPlayers(t *testing.T) {
	addresses := createAddresses(2)
	fatal := make(chan error)
	for i := 0; i < 2; i++ {
		go func() {
			p := Player{
				Rank:      i,
				Addresses: addresses,
			}
			time.Sleep(time.Second * time.Duration(i+1))
			recv, err := p.Broadcast([]byte{'0'}, 0)
			if err != nil {
				fatal <- err
				return
			}
			if recv[0] != '0' {
				fatal <- fmt.Errorf("from player %d: expected %s, actual %s", i, "0", recv)
				return
			}
			time.Sleep(time.Second * time.Duration(i+1))
			recv, err = p.Broadcast([]byte{'1'}, 1)
			if err != nil {
				fatal <- err
				return
			}
			if recv[0] != '1' {
				fatal <- fmt.Errorf("from player %d: expected %s, actual %s", i, "1", recv)
				return
			}
			fatal <- nil
		}()
	}
	for i := 0; i < 2; i++ {
		err := <-fatal
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestBroadcastBarrier(t *testing.T) {
	n := 10
	addresses := createAddresses(n)
	fatal := make(chan error, n)
	clocks := make(chan int, 2*n)
	for i := 0; i < n; i++ {
		go func(i int) {
			p := Player{
				Rank:      i,
				Addresses: addresses,
			}
			time.Sleep(time.Millisecond * 100 * time.Duration(p.Rank))
			clocks <- 0
			_, err := p.Broadcast(nil, 0)
			time.Sleep(time.Millisecond * 100 * time.Duration(p.Rank))
			clocks <- 1
			if err != nil {
				fatal <- err
				return
			}
			fatal <- nil
		}(i)
	}
	for i := 0; i < n; i++ {
		err := <-fatal
		if err != nil {
			t.Fatal(err)
		}
	}
	close(clocks)
	prev := 0
	for time := range clocks {
		if prev > time {
			t.Fatalf("clocks out of sync: prev %d, time %d", prev, time)
		} else {
			prev = time
		}
	}
}

func TestAllToAllBarrier(t *testing.T) {
	n := 10
	addresses := createAddresses(n)
	fatal := make(chan error, n)
	clocks := make(chan int, 2*n)
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			p := Player{
				Rank:      i,
				Addresses: addresses,
			}
			time.Sleep(time.Millisecond * 100 * time.Duration(p.Rank))
			clocks <- 0
			_, err := p.AllToAll([]byte{})
			time.Sleep(time.Millisecond * 100 * time.Duration(p.Rank))
			if err != nil {
				fatal <- err
				return
			}
			clocks <- 1
		}(i)
	}
	wg.Wait()
	close(fatal)
	for err := range fatal {
		t.Error(err)
	}
	close(clocks)
	prev := 0
	for time := range clocks {
		if prev > time {
			t.Fatalf("clocks out of sync: prev %d, time %d", prev, time)
		} else {
			prev = time
		}
	}
}
