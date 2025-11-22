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
			discover, err := NewWithPortRange(fmt.Sprint(i), 9000, 9010, 2)
			if err != nil {
				fatal <- err
				return
			}
			set := make(map[string]struct{})
			for range n-1 {
				entry := <-discover.Entries
				t.Logf("from node %d: %s", i, entry)
				set[entry.Info] = struct{}{}
			}
			for j := range n {
				if j == i {
					continue
				}
				if _, ok := set[fmt.Sprint(j)]; !ok {
					fatal <- fmt.Errorf("node %d did not find entry %d",i, j)
					return
				}
			}
			time.Sleep(5*time.Second)
			fatal <- discover.Close()
		}()
	}
	for range n {
		if err := <-fatal; err != nil {
			t.Fatal(err)
		}
	}
}
