package common

import (
	"fmt"
	"math"
	"strconv"
)

// Similar to humanize.Bytes, but with more detail

func logn(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}

func NumBytesDisplayString(s uint64) string {
	sizes := []string{"B", "kB", "MB", "GB"}
	if s < 10 {
		return fmt.Sprintf("%d B", s)
	}
	base := 1000.0
	f := float64(s)
	e := math.Floor(logn(f, base))
	ix := int(e)
	if ix > len(sizes) {
		ix = len(sizes)-1
	}
	val := math.Floor(f/math.Pow(base, e)*10.0+0.5) / 10
	decimals := 0
	switch ix {
	case 2: decimals = 1	// MB
	case 3: decimals = 3	// GB
	}

	return fmt.Sprintf("%." + strconv.Itoa(decimals) + "f %s", val, sizes[ix])
}

