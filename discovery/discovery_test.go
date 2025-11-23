package discovery

import (
	"fmt"
	"testing"
	"time"
)

func TestDiscover(t *testing.T) {
	n := 5
	fatal := make(chan error)
	for i := range n {
		go func() {
			discover, err := New(fmt.Sprint(i), 53551,
				WithIntervalBetweenAnnouncements(200*time.Millisecond),
			)
			if err != nil {
				fatal <- err
				return
			}
			set := make(map[string]struct{})
			for j := 0; j < n-1; {
				entry := <-discover.Entries
				t.Logf("from node %d: %s", i, entry)
				if _, ok := set[entry.Info]; !ok {
					j++
				}
				set[entry.Info] = struct{}{}
			}
			for j := range n {
				if j == i {
					continue
				}
				if _, ok := set[fmt.Sprint(j)]; !ok {
					fatal <- fmt.Errorf("node %d did not find entry %d", i, j)
					return
				}
			}
			fatal <- nil
		}()
	}
	for range n {
		if err := <-fatal; err != nil {
			t.Fatal(err)
		}
	}
}

func TestClose(t *testing.T) {
	n := 5
	fatal := make(chan error)
	for i := range n {
		go func() {
			discover, err := New(fmt.Sprint(i), 53552)
			if err != nil {
				fatal <- err
				return
			}
			time.Sleep(500 * time.Millisecond)
			if err := discover.Close(); err != nil {
				fatal <- err
				return
			}
			fatal <- nil
		}()
	}
	for range n {
		if err := <-fatal; err != nil {
			t.Fatal(err)
		}
	}
}
