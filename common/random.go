package common

import (
	crand "crypto/rand"
	"math/big"
	mrand "math/rand"
)

func RandomInt64( lower, upperExclusive int64) int64 {
	nBig, err := crand.Int(crand.Reader, big.NewInt(upperExclusive))
	if err == nil {
		return nBig.Int64() + lower
	}
	return mrand.Int63n(upperExclusive)
}

func RandomInt( lower, upperExclusive int) int {
	r := RandomInt64(int64(lower), int64(upperExclusive))
	return int(r)
}

// Returns range of n
func RandomRange(n, lower, upperExclusive int) []int {
	if upperExclusive - lower < n {
		panic("RandomRange")
	}
	m := make(map[int]bool)
	for len(m) < n {
		m[RandomInt(lower, upperExclusive)] = true
	}
	r := make([]int,n)
	j := 0
	for i, _ := range m {
		r[j] = i
		j++
	}
	return r
}

// RandomIntRange return n random ints in the range 0 to n-1
func RandomIntRange(n int) []int {
	return RandomRange(n, 0, n)
}

func RandomIntFromSlice( a []int) int {
	return a[RandomInt(0, len(a))]
}

// RandomStringFromSlice returns a random string and removes it from the slice
// Empty string is returned once the slice is empty
func RandomStringFromSlice(a *[]string) string {
	ac := *a
	l := len(ac)
	if l == 0 {
		return ""
	}
	i := RandomInt(0, l)
	r := ac[i]
	// Erase
	ac[i] = ac[l-1]
	*a = ac[:l-1]
	return r
}

// FillRandom fills a byte slice with random values
func FillRandom(p []byte) {
	for i := 0; i < len(p); i += 7 {
		val := mrand.Int63()
		for j := 0; i+j < len(p) && j < 7; j++ {
			p[i+j] = byte(val)
			val >>= 8
		}
	}
}

