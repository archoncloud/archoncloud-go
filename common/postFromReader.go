package common

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"io"
	"net/http"
	"strings"
	"sync"
)

// PostFromReader doest a POST to the targetUrl with data from r and reports an error, if any
func PostFromReader(targetUrl string, r io.Reader, contentType string) (err error) {
	client := http.Client{
		Timeout: 0,	// 0 means no timeout
	}
	resp, err := client.Post(targetUrl, contentType, r)
	if err == nil {
		_, err = GetResponse(resp)
		resp.Body.Close()
	}
	return
}

func PostFromReaderWithProgress(targetUrl string, r io.Reader, contentType string, progress *ByteProgress) (err error) {
	var progressReader io.Reader
	if progress == nil {
		progressReader = r
	} else {
		pw := ProgressWriter{
			Progress: progress,
		}
		progressReader = io.TeeReader(r,&pw)
	}
	err = PostFromReader(targetUrl, progressReader, contentType)
	return
}

// ByteProgress is needed when aggregating data from code running in parallel
type ByteProgress struct {
	mux				sync.Mutex
	what			string
	show			bool
	Total			uint64
}

func clearLine() {
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	fmt.Printf("\r%s", strings.Repeat(" ", 80))
}

func (bp *ByteProgress) showProgress() {
	clearLine()
	fmt.Printf("\rUploading %s... %s", bp.what, NumBytesDisplayString(bp.Total))
}

func NewByteProgress(prefix string, total uint64) *ByteProgress {
	return &ByteProgress{Total: 0, what: prefix, show: total >= 6*humanize.MByte}
}

func (bp *ByteProgress) SetPrefix(pref string) {
	if bp.show {
		bp.mux.Lock()
		bp.what = pref
		bp.showProgress()
		bp.mux.Unlock()
	} else {
		fmt.Println(pref)
	}
}

func (bp *ByteProgress) Progress(n uint64) {
	if bp.show {
		bp.mux.Lock()
		bp.Total += n
		bp.showProgress()
		bp.mux.Unlock()
	}
}

func (bp *ByteProgress) End() {
	if bp.show {
		clearLine()
	}
}

// ProgressWriter counts the number of bytes written to it. It implements to the io.Reader
// interface and we can pass this into io.TeeReader() which will report progress on each
// Read cycle.
type ProgressWriter struct {
	Progress	*ByteProgress
}

// Write only collects the number of bytes and calls Progress. It does not actually write
func (pw *ProgressWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	pw.Progress.Progress(uint64(n))
	return n, nil
}
