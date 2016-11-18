package rand

import (
	"fmt"
	"math/rand"
	"time"
)

var random = rand.NewSource(time.Now().UnixNano())

// RandHex produces a hex string containing n random bytes; thus of length 2*n.
func RandHex(n int) string {
	buf := make([]byte, 2*n)
	digitsRem := 0
	var randomNumber int64
	for i := 0; i < n*2; i++ {
		if digitsRem == 0 {
			digitsRem = 7
			randomNumber = random.Int63()
		}
		buf[i] = (fmt.Sprintf("%x", randomNumber&0x0F))[0]
		randomNumber >>= 4
		digitsRem--
	}
	return string(buf)
}
