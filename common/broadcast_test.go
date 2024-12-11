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
			p, err := NewPlayer(i, addresses)
			if err != nil {
				fatal <- err
				return
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
	addresses := createAddresses(10)
	root := 3
	fatal := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			p, err := NewPlayer(i, addresses)
			if err != nil {
				fatal <- err
				return
			}
			a := strconv.Itoa(10 * i)
			time.Sleep(time.Millisecond * 100 * time.Duration(p.Rank))
			recv, err := p.Broadcast(a, root)
			time.Sleep(time.Millisecond * 100 * time.Duration(p.Rank))
			if err != nil {
				fatal <- err
				return
			}
			actual, err := strconv.Atoi(recv)
			if err != nil {
				fatal <- err
				return
			}
			if actual != root*10 {
				fatal <- fmt.Errorf("expected %d, actual %d", actual, root*10)
				return
			}
			fatal <- nil
		}(i)
	}
	for i := 0; i < 10; i++ {
		err := <-fatal
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestBroadcastBarrier(t *testing.T) {
	addresses := createAddresses(10)
	fatal := make(chan error, 10)
	clocks := make(chan int, 20)
	for i := 0; i < 10; i++ {
		go func(i int) {
			p, err := NewPlayer(i, addresses)
			if err != nil {
				fatal <- err
				return
			}
			time.Sleep(time.Millisecond * 100 * time.Duration(p.Rank))
			clocks <- 0
			_, err = p.Broadcast("", 0)
			time.Sleep(time.Millisecond * 100 * time.Duration(p.Rank))
			clocks <- 1
			if err != nil {
				fatal <- err
				return
			}
			fatal <- nil
		}(i)
	}
	for i := 0; i < 10; i++ {
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
	addresses := createAddresses(10)
	fatal := make(chan error, 10)
	clocks := make(chan int, 20)
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer wg.Done()
			p, err := NewPlayer(i, addresses)
			if err != nil {
				fatal <- err
				return
			}
			time.Sleep(time.Millisecond * 100 * time.Duration(p.Rank))
			clocks <- 0
			_, err = p.AllToAll([]byte{})
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

func TestAllToAllBarrierRepeated(t *testing.T) {
	for i := 0; i < 100; i++ {
		t.Log(i)
		TestAllToAllBarrier(t)
	}
}
