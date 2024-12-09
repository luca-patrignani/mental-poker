package common

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAllToAll(t *testing.T) {
	n := 10
	addresses := []string{}
	for i := 0; i < n; i++ {
		addresses = append(addresses, fmt.Sprintf("127.0.0.1:111%d", i))
	}
	fatal := make(chan error, 3*n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			p := Player{Rank: i, Addresses: addresses}
			actual, err := p.AllToAll(strconv.Itoa(i))
			if err != nil {
				fatal <- err
				return
			}
			if len(actual) != n {
				fatal <- fmt.Errorf("expected list of length %d, %v given", n, actual)
				return
			}
			for j := 0; j < n; j++ {
				if strings.Compare(strconv.Itoa(j), actual[j]) != 0 {
					fatal <-fmt.Errorf("expected %d, actual %v", j, actual[j])
					return
				}
			}
		}()
	}
	wg.Wait()
	close(fatal)
	for err := range fatal {
		t.Error(err)
	}
}

func Test3(t *testing.T) {
	n := 3
	addresses := []string{}
	for i := 0; i < n; i++ {
		addresses = append(addresses, fmt.Sprintf("127.0.0.1:111%d", i))
	}
	b1 := Player{Rank: 0, Addresses: addresses}
	b2 := Player{Rank: 1, Addresses: addresses}
	b3 := Player{Rank: 2, Addresses: addresses}
	go b1.AllToAll("a1")
	go b3.AllToAll("a3")
	actual, err := b2.AllToAll("a2")
	if err != nil {
		t.Fatal(err)
	}
	if len(actual) != 3 {
		t.Fatal()
	}
	if actual[0] != "a1" || actual[2] != "a3" {
		t.Fatal()
	}
}

func TestBroadcast(t *testing.T) {
	addresses := []string{}
	for i := 0; i < 10; i++ {
		addresses = append(addresses, fmt.Sprintf("127.0.0.1:1111%d", i))
	}
	root := 3
	fatal := make(chan error, 10)
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer wg.Done()
			p := Player{Rank: i, Addresses: addresses}
			a := strconv.Itoa(10 * i)
			time.Sleep(time.Millisecond * time.Duration(rand.Int31n(1000)))
			recv, err := p.Broadcast(a, root)
			time.Sleep(time.Millisecond * time.Duration(rand.Int31n(1000)))
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
		}(i)
	}
	wg.Wait()
	close(fatal)
	for err := range fatal {
		t.Error(err)
	}
}

func TestBroadcastBarrier(t *testing.T) {
	addresses := []string{}
	for i := 0; i < 10; i++ {
		addresses = append(addresses, fmt.Sprintf("127.0.0.1:111%d", i))
	}
	fatal := make(chan error, 10)
	clocks := make(chan int, 20)
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer wg.Done()
			p := Player{Rank: i, Addresses: addresses}
			time.Sleep(time.Millisecond * time.Duration(rand.Int31n(1000)))
			clocks <- 0
			_, err := p.Broadcast("", 0)
			time.Sleep(time.Millisecond * time.Duration(rand.Int31n(1000)))
			clocks <- 1
			if err != nil {
				fatal <- err
				return
			}
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

func TestAllToAllBarrier(t *testing.T) {
	addresses := []string{}
	for i := 0; i < 10; i++ {
		addresses = append(addresses, fmt.Sprintf("127.0.0.1:111%d", i))
	}
	fatal := make(chan error, 10)
	clocks := make(chan int, 20)
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer wg.Done()
			p := Player{Rank: i, Addresses: addresses}
			time.Sleep(time.Millisecond * time.Duration(rand.Int31n(1000)))
			clocks <- 0
			_, err := p.AllToAll("")
			time.Sleep(time.Millisecond * time.Duration(rand.Int31n(1000)))
			clocks <- 1
			if err != nil {
				fatal <- err
				return
			}
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
