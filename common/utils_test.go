package common

import (
	"bytes"
	"fmt"
	"gotest.tools/assert"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestNumLines(t *testing.T) {
	const toWrite = 20
	sb := strings.Builder{}
	for i := 0; i < toWrite; i++ {
		sb.WriteString(fmt.Sprintf("Line %d\n", i ))
	}
	numLines, err := NumLines(bytes.NewBufferString(sb.String()))
	if err != nil {
		t.Error(err)
	}
	if numLines != toWrite {
		t.Errorf("Got %d lines. Expected %d", numLines, toWrite)
	}
}

func TestWriteLastLines(t *testing.T) {
	const expectedLastLines = 10
	tempFileName := writeNumLines(150)
	if tempFileName == "" {
		t.Errorf("Could not write temp file")
		return
	}
	defer os.Remove(tempFileName)
	lastBytes := &bytes.Buffer{}
	WriteLastLines(lastBytes, tempFileName, expectedLastLines)
	lastLines, err := NumLines(lastBytes)
	if err != nil {
		t.Error(err)
	}
	if lastLines != expectedLastLines {
		t.Errorf("Got %d lines. Expected %d", lastLines, expectedLastLines)
	}
}

func writeNumLines(numLines int) string {
	tempFile, err := ioutil.TempFile("", "archon")
	if err != nil {
		return ""
	}
	defer tempFile.Close()
	for i := 0; i < numLines; i++ {
		tempFile.WriteString( fmt.Sprintf("Line %d\n", i ))
	}
	return tempFile.Name()
}

func TestRoundUp(t *testing.T) {
	r := RoundUp( 15, 6)
	assert.Equal(t, uint64(18), r, "RoundUp" )
}
