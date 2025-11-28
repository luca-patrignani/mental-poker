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
			discover := Discover{
				Info:                         []byte(fmt.Sprint(i)),
				IntervalBetweenAnnouncements: 200 * time.Millisecond,
				Port:                         53552,
			}
			if err := discover.Start(); err != nil {
				fatal <- err
				return
			}
			set := make(map[string]struct{})
			for j := 0; j < n-1; {
				entry := <-discover.Entries
				t.Logf("from node %d: %s", i, entry)
				if time.Since(entry.Time) < 0 {
					fatal <- fmt.Errorf("from node %d: time out of clock", i)
					return
				}
				info := string(entry.Info)
				if _, ok := set[info]; !ok {
					j++
				}
				set[info] = struct{}{}
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
			discover := Discover{
				Info: []byte(fmt.Sprint(i)),
				Port: 53553,
			}
			if err := discover.Start(); err != nil {
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
