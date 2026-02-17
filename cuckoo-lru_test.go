package coolru

import "fmt"
import "encoding/binary"
import "testing"
import "github.com/stretchr/testify/require"

type Obj int64
func (o Obj) MarshalBinary() ([]byte, error) {
        buf := make([]byte, 8)
        binary.BigEndian.PutUint64(buf, uint64(o))
        return buf, nil
}

func TestBasic(t *testing.T) {
        filt, _ := New(1<<16, 0.0001, 1<<12)
        for i := range (1<<12) {
      	  filt.Add(Obj(i))
        }
	falsePositives := 0
        for i := 1<<12; i < (1<<20) && falsePositives < (1<<12); i++ {
	  io := Obj(i)
      	  oops, _ := filt.Check(io)
      	  if oops {
		  falsePositives++
		  // High probability of at least one false positive, likely more
      		  filt.MarkFalsePositive(io)
      		  unoops, _ := filt.Check(io)
		  errMsg := fmt.Sprintf("Oopsed on %d, marked FP, but now it returns %t\n", i, unoops)
		  require.Equal(t, unoops, false, errMsg)
      	  }
        }
        for i := range (1<<12) {
		io := Obj(i)
		res, _ := filt.Check(io)
		require.Equal(t, res, true, fmt.Sprintf("Expected true for value %d", i))
	}
        for i := 1<<12; i < (1<<20); i++ {
		io := Obj(i)
		res, _ := filt.Check(io)
		require.Equal(t, res, false, fmt.Sprintf("Expected false for value %d", i))
	}
}
