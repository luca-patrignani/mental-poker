package common

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
)

func Test(t *testing.T) {
	b1 := Player{}
	b2 := Player{}
	go b1.BroadcastAllToAll("a1")
	actual, err := b2.BroadcastAllToAll("a2")
	if err != nil {
		t.Fatal(err)
	}
	if len(actual) != 2 {
		t.Fatal()
	}
	if actual[0] != "a1" {
		t.Fatal()
	}
	if actual[1] != "a2" {
		t.Fatal()
	}
}

func Test3(t *testing.T) {
	b1 := Player{}
	b2 := Player{}
	b3 := Player{}
	go b1.BroadcastAllToAll("a1")
	go b3.BroadcastAllToAll("a3")
	actual, err := b2.BroadcastAllToAll("a2")
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

func Test4(t *testing.T) {
	addresses := []string{}
	for i := 0; i < 10; i++ {
		addresses = append(addresses, fmt.Sprintf("127.0.0.1:111%d", i))
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
			recv, err := p.Broadcast(a, root)
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
