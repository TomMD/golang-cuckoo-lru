/*
Cuckoo+LRU (blur) is a basic marriage between a cuckoo filter to reduce hits and an LRU to ensure popular addresses that are known false-positives do not return true.

  import blur "github.com/tommd/bloom-lru"

  type Obj int64
  func (o Obj) MarshalBinary() ([]byte, error) {
	  buf := make([]byte, 8)
	  binary.BigEndian.PutUint64(buf, uint64(o))
	  return buf, nil
  }
  func main() {
	  blur, _ := blur.New(1<<16, 0.0001, 1<<10)
	  for i := range (1<<12) {
		  blur.Add(Obj(i))
	  }
	  for i := 1<<12; i < (1<<20); i++ {
		  oops, _ := blur.Check(Obj(i))
		  if oops {
			  blur.MarkFalsePositive(Obj(i))
			  unoops, _ := blur.Check(Obj(i))
			  fmt.Printf("Oopsed on %d but now it returns %t\n", i, unoops)
			  break;
		  }
	  }
  }
*/
package bloom_lru

import (
	"encoding"
	"math"
	"sync"

	cuckoo "github.com/linvon/cuckoo-filter"
	lru "github.com/hashicorp/golang-lru"
)

const (
	tagsPerBucket = 4
)

type BloomLRU struct {
	filter  *cuckoo.Filter
	arc     *lru.ARCCache
	lock    sync.RWMutex
}

func New(capacity uint, falsePositives float64, lruSize int) (BloomLRU, error) {
	// Calculate fingerprint bits needed for target false positive rate
	// FP rate ≈ 1/2^bitsPerItem, so bitsPerItem = ceil(log2(1/fp))
	bitsPerItem := uint(math.Ceil(math.Log2(1.0 / falsePositives)))
	if bitsPerItem < 4 {
		bitsPerItem = 4
	}
	if bitsPerItem > 32 {
		bitsPerItem = 32
	}

	f := cuckoo.NewFilter(tagsPerBucket, bitsPerItem, capacity, cuckoo.TableTypePacked)
	a, err := lru.NewARC(lruSize)
	if err != nil {
		return BloomLRU{}, err
	}
	return BloomLRU{
		filter: f,
		arc:    a,
	}, nil
}

func (f *BloomLRU) MarshalBinary() ([]byte, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.filter.Encode(), nil
}

func (f *BloomLRU) UnmarshalBinary(data []byte) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	decoded, err := cuckoo.Decode(data)
	if err != nil {
		return err
	}
	f.filter = decoded
	return nil
}

func (f *BloomLRU) Add(elem encoding.BinaryMarshaler) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	bytes, err := elem.MarshalBinary()
	if err != nil {
		return err
	}
	f.filter.Add(bytes)
	return nil
}

func (f *BloomLRU) MarkFalsePositive(elem any) {
	f.arc.Add(elem, true)
}

func (f *BloomLRU) Check(elem encoding.BinaryMarshaler) (bool, error) {
	bytes, err := elem.MarshalBinary()
	if err != nil {
		return false, err
	}
	f.lock.RLock()
	defer f.lock.RUnlock()
	filterRes := f.filter.Contain(bytes)
	if filterRes {
		_, isFalsePositive := f.arc.Get(elem)
		return !isFalsePositive, nil
	}
	return filterRes, nil
}
