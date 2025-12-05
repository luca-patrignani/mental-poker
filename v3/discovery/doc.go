// Package discovery provides a lightweight UDP multicast-based service discovery mechanism.

// This package allows multiple instances to announce information over a local network
// using multicast UDP and discover announcements from other peers. Typical usage:

// 	d := &discovery.Discover{
// 		Info:                         []byte("my-service-info"),
// 		Port:                         5353,
// 		IntervalBetweenAnnouncements: 5 * time.Second,
// 	}
// 	if err := d.Start(); err != nil {
// 		log.Fatal(err)
// 	}
// 	defer d.Close()

// 	for entry := range d.Entries {
// 		fmt.Printf("Discovered: %s at %v\n", entry.Info, entry.Time)
// 	}

// Behavior:
//   - Announcements are sent via UDP multicast to 239.0.0.1 on the specified port.
//   - Each instance uses a random 8-byte key to identify its own packets and filter them out.
//   - Discovered entries are delivered on the Entries channel.
//   - The implementation panics on unrecoverable network errors.
package discovery