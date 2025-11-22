package discovery

import "testing"

func TestDiscover(t *testing.T) {
	n := 5
	fatal := make(chan error)

	for i := range n {
		go func() {
			_ = i
			discover, err := New("=== Mental Poker ===")
			if err != nil {
				fatal <- err
				return
			}
			for range n-1 {
				entry := <-discover.Entries
				t.Log(entry)
				if entry.Info != "=== Mental Poker ===" {
					fatal <- err
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
