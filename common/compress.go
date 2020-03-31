package common

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
)

func CompressString(s string) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	gz.Write([]byte(s))
	gz.Flush()
	gz.Close()
	return b.Bytes()
}

func UnCompressString(b []byte) string {
	rdata := bytes.NewReader(b)
	r,_ := gzip.NewReader(rdata)
	s, _ := ioutil.ReadAll(r)
	return string(s)
}
