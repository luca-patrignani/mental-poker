package main

import (
	"fmt"
	"testing"
	"time"
)

func TestPinger(t *testing.T) {
	n := 5
	fatal := make(chan error)
	for i := range n {
		go func() {
			p, err := NewPinger(
				Info{Name: fmt.Sprint(i)},
				time.Millisecond,
			)
			if err != nil {
				fatal <- err
				return
			}
			if err := p.Start(); err != nil {
				fatal <- err
				return
			}
			time.Sleep(time.Second)
			found := map[string]struct{}{}
			playerStatues := p.PlayersStatus()
			for info := range playerStatues {
				if _, ok := found[info.Name]; ok {
					fatal <- fmt.Errorf("from node %d, duplicated player %s found", i, info.Name)
					return
				}
				found[info.Name] = struct{}{}
			}
			for j := range n {
				_, ok := found[fmt.Sprint(j)]
				if i == j {
					if ok {
						fatal <- fmt.Errorf("node %d found itself", i)
						return
					}
					continue
				}
				if !ok {
					fatal <- fmt.Errorf("node %d did not find %d", i, j)
					return
				}

			}
			fatal <- p.Close()
		}()
	}
	for range n {
		if err := <-fatal; err != nil {
			t.Fatal(err)
		}
	}
}
