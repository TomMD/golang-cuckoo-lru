# Cuckoo-lru (CooLRU)

A cuckoo filter with an LRU cache to track known false positives.
The LRU exists to catch popular elements that are false positives in the
cuckoo filter from wreaking havoc for any later processing.

While the underlying hashicorp ARC (LRU) is itself threadsafe, a mutex is
used to ensure safety of the cuckoo filter.

## Example

```go
package main

import (
	"encoding/binary"
	"fmt"

	coolru "github.com/tommd/golang-cuckoo-lru"
)

type Key int64

func (k Key) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(k))
	return buf, nil
}

func main() {
	// Create filter with capacity 1000, 0.01% false positive rate, and LRU size 100
	f, _ := coolru.New(1000, 0.0001, 100)

	// Add an element
	f.Add(Key(42))

	// Check returns true for added elements
	found, _ := f.Check(Key(42))
	fmt.Println("After Add:", found) // true

	// Mark as false positive
	f.MarkFalsePositive(Key(42))

	// Check now returns false
	found, _ = f.Check(Key(42))
	fmt.Println("After MarkFalsePositive:", found) // false

	// Serialize (LRU data is NOT included)
	data, _ := f.MarshalBinary()

	// Deserialize into a new filter (only LRU size needed)
	f2, _ := coolru.NewFromBytes(data, 100)

	// LRU is never serialized, so false positive mark is gone
    // in practice this will impact application warm up
	found, _ = f2.Check(Key(42))
	fmt.Println("After Deserialize:", found) // true
}
```
