/*
Cuckoo+LRU (filt) is a basic marriage between a cuckoo filter to reduce hits and an LRU to ensure popular addresses that are known false-positives do not return true.

  import coolru "github.com/tommd/golang-cuckoo-lru"

  type Obj int64
  func (o Obj) MarshalBinary() ([]byte, error) {
	  buf := make([]byte, 8)
	  binary.BigEndian.PutUint64(buf, uint64(o))
	  return buf, nil
  }
  func main() {
	  filt, _ := coolru.New(1<<16, 0.0001, 1<<10)
	  for i := range (1<<12) {
		  filt.Add(Obj(i))
	  }
	  for i := 1<<12; i < (1<<20); i++ {
		  oops, _ := filt.Check(Obj(i))
		  if oops {
			  filt.MarkFalsePositive(Obj(i))
			  unoops, _ := filt.Check(Obj(i))
			  fmt.Printf("Oopsed on %d but now it returns %t\n", i, unoops)
			  break;
		  }
	  }
  }
*/
package coolru

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

type CooLRU struct {
	filter  *cuckoo.Filter
	arc     *lru.ARCCache
	lock    sync.RWMutex
}

func New(capacity uint, falsePositives float64, lruSize int) (CooLRU, error) {
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
		return CooLRU{}, err
	}
	return CooLRU{
		filter: f,
		arc:    a,
	}, nil
}

func NewFromBytes(data []byte, lruSize int) (CooLRU, error) {
	a, err := lru.NewARC(lruSize)
	if err != nil {
		return CooLRU{}, err
	}
	f, err := cuckoo.Decode(data)
	if err != nil {
		return CooLRU{}, err
	}
	return CooLRU{filter: f, arc: a}, nil
}

func (f *CooLRU) MarshalBinary() ([]byte, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.filter.Encode()
}

func (f *CooLRU) UnmarshalBinary(data []byte) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	decoded, err := cuckoo.Decode(data)
	if err != nil {
		return err
	}
	f.filter = decoded
	return nil
}

func (f *CooLRU) Add(elem encoding.BinaryMarshaler) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	bytes, err := elem.MarshalBinary()
	if err != nil {
		return err
	}
	f.filter.Add(bytes)
	return nil
}

func (f *CooLRU) MarkFalsePositive(elem any) {
	f.arc.Add(elem, true)
}

func (f *CooLRU) Check(elem encoding.BinaryMarshaler) (bool, error) {
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
