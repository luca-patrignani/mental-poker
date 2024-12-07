package common

import (
	"strconv"
	"testing"
	"sync"
)

func Test(t *testing.T) {
	b1 := Player{}
	b2 := Player{}
	go b1.BroadcastAllToAll("a1")
	actual := b2.BroadcastAllToAll("a2")
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
	actual := b2.BroadcastAllToAll("a2")
	if len(actual) != 3 {
		t.Fatal()
	}
	if actual[0] != "a1" || actual[2] != "a3" {
		t.Fatal()
	}
}

func Test4(t *testing.T) {
	root := 0
	fatal := make(chan error, 10)
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer wg.Done()
			b := Player{MyRank: i}
			a := strconv.Itoa(10 * i)
			recv, err := b.Broadcast(a, root)
			if err != nil {
				fatal <- err
			}
			if recv != strconv.Itoa(root*10) {
				fatal <- err
			}
		}(i)
	}
	wg.Wait()
	close(fatal)
	for err := range fatal {
		t.Error(err)
	}
}
