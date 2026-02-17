# bloom-lru

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

	blur "github.com/tommd/bloom-lru"
)

type Key int64

func (k Key) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(k))
	return buf, nil
}

func main() {
	// Create filter with capacity 1000, 0.01% false positive rate, and LRU size 100
	f, _ := blur.New(1000, 0.0001, 100)

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

	// Deserialize into a new filter
	f2, _ := blur.New(1000, 0.0001, 100)
	f2.UnmarshalBinary(data)

	// LRU was not serialized, so false positive mark is gone
	found, _ = f2.Check(Key(42))
	fmt.Println("After Deserialize:", found) // true
}
```
