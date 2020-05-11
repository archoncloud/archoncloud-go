package common

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// DownloadFile will download a file from a url. Assumes that url is a file download url
func DownloadFile(w io.Writer, url string) error {
	client := http.Client{
		Timeout:0, // 0 means no timeout
	}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s\n", resp.Status)
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

// WriteLastLines writes the last numLines from filepath to w
func WriteLastLines(w io.Writer, filepath string, numLines int) {
	fileHandle, err := os.Open(filepath)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	defer fileHandle.Close()

	var cursor int64 = 0
	stat, _ := fileHandle.Stat()
	filesize := stat.Size()
	toRead := numLines
	for {
		cursor--
		if cursor == -filesize { // stop if we are at the beginning
			break
		}
		_, err = fileHandle.Seek(cursor, io.SeekEnd)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		char := make([]byte, 1)
		_, err = fileHandle.Read(char)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		if char[0] == '\n' {
			// end of a line
			toRead--
			if toRead <= 0 {
				break
			}
		}
	}
	offset := filesize + cursor
	buf := make([]byte, -cursor)
	_, err = fileHandle.ReadAt(buf,offset)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(buf)
}

func NumLines(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}

// RoundUp rounds up. Amazing this does not exist in the standard packages
func RoundUp(numToRound, multiple uint64) uint64 {
	return ((numToRound + multiple - 1) / multiple) * multiple
}

func DivideRoundUp(numerator, divisor uint64) uint64 {
	return ((numerator + divisor - 1) / divisor)
}

func MegaBytes(numBytes int64) int64 {
	return int64(DivideRoundUp(uint64(numBytes),Mega))
}

func ExtendedSliceToMultipleOf(multiple int, input []byte) []byte {
	l := len(input)
	rem := l % multiple
	if rem == 0 {
		return input
	} else {
		output := append(input, make([]byte, multiple-rem )...)
		return output
	}
}

// Max returns the larger of x or y.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// Min returns the smaller of x or y.
func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

// IsLegalFilePath checks for forbidden characters
func IsLegalFilePath(path string) error {
	const invalidFilePathChars = `#~*<>:”?`
	if strings.ContainsAny(path, invalidFilePathChars ) {
		return fmt.Errorf("%s cannot contain the following characters: %s", path,  invalidFilePathChars)
	}
	const maxPath = 200
	if len(path) > maxPath {
		return fmt.Errorf("%s exceeds max len(%d)", path, maxPath)
	}
	return nil
}

// IsLegalUserName checks for forbidden characters
func IsLegalUserName(userName string) error {
	const invalidUserNameChars = `/\.#@~*<>:"?`
	if strings.ContainsAny(userName, invalidUserNameChars ) {
		return fmt.Errorf("username cannot contain the following characters: %s", invalidUserNameChars)
	}
	return nil
}

// PromptForInput prompts on stdin and reads a line. Empty string is returned on error
func PromptForInput(prompt string) string {
	fmt.Println(prompt)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func Yes(prompt string) bool {
	text := PromptForInput(prompt + "?[y/n]:")
	return text != "" && (text[0] == 'y' || text[0] == 'Y')
}

// Abort does nothing if err is nil, otherwise logs the error and exits
func Abort(err error) {
	if err != nil {
		msg := strings.TrimLeft(err.Error(), " ")
		if LogError == nil {
			fmt.Println(msg)
		} else {
			msg = strings.TrimPrefix(msg, "Error")
			msg = strings.TrimPrefix(msg, "error")
			LogError.Println(msg)
		}
		os.Exit(1)
	}
}

func AbortWithString(msg string) {
	Abort(errors.New(msg))
}

func BytesToString(data []byte) string {
	return hexutil.Encode(data)
}

// RawBytesToString returns a string without the 0x prefix
func RawBytesToString(data []byte) string {
	return strings.TrimPrefix(hexutil.Encode(data),"0x")
}

func StringToBytes(s string) []byte {
	if !strings.HasPrefix(s, "0x") {
		s = "0x" + s
	}
	b, err := hexutil.Decode(s)
	if err == nil {return b}
	return nil
}

func StringToEthAddress(s string) (ea [20]byte) {
	copy(ea[:], StringToBytes(s))
	return
}

func I64ToA(i int64) string {
	return fmt.Sprintf("%d", i);
}

func SeparatedStringList(separator string, items ...interface{}) string {
	var buf []string
	for _, s := range items {
		var str string
		if v, ok := s.(string); ok {
			str = v
		} else if v, ok := s.(int); ok {
			str = strconv.Itoa(v)
		} else if v, ok := s.(int64); ok {
			str = I64ToA(v)
		} else if v, ok := s.(float64); ok {
			str = strconv.FormatFloat(v, 'g',-1,64 )
		} else if v, ok := s.([]string); ok {
			buf = append(buf,v...)
			continue
		} else {
			panic("SeparatedStringList")
		}
		buf = append(buf, str)
	}
	return strings.Join(buf, separator)
}

func InvalidArgs(msg string) {
	Abort( fmt.Errorf("%s. Type --help", msg))
}

func NeedArgs(arg string) {
	Abort(fmt.Errorf("You need to provide the -%s argument. Type -help\n", arg))
}

func EraseInt(a []int, x int) []int {
	for i, n := range a {
		if x == n {
			a[i] = a[len(a)-1]
			return a[:len(a)-1]
		}
	}
	return a
}

func EraseString(a []string, x string) []string {
	for i, n := range a {
		if x == n {
			a[i] = a[len(a)-1]
			return a[:len(a)-1]
		}
	}
	return a
}

func CloneStringSlice(a []string) []string {
	clone := append([]string(nil), a...)
	return clone
}

func ReadExactly( n int, r io.Reader ) (data []byte, err error) {
	data = make([]byte, n)
	read, err := r.Read(data)
	if err != nil {return}
	if n != read {
		err = fmt.Errorf("expected %d bytes", n)
	}
	return
}

func ReadByte(r io.Reader) (v byte, err error) {
	buf, err := ReadExactly(1, r)
	if err == nil {
		v = buf[0]
	}
	return
}

func ReadBigEndianUint32(r io.Reader) (v uint32, err error) {
	buf, err := ReadExactly(4, r)
	if err == nil {
		v = binary.BigEndian.Uint32(buf)
	}
	return
}

func ReadBigEndianUint64(r io.Reader) (v uint64, err error) {
	buf, err := ReadExactly(8, r)
	if err == nil {
		v = binary.BigEndian.Uint64(buf)
	}
	return
}

func ReadBigEndianInt32(r io.Reader) (v int32, err error) {
	buf, err := ReadExactly(4, r)
	if err == nil {
		v = int32(binary.BigEndian.Uint32(buf))
	}
	return
}

func StringKeysOf(m map[string]bool) []string {
	keys := []string{}
	for k, _ := range m {
		keys = append(keys, k)
	}
	return keys
}

func ReadBigEndianInt64(r io.Reader) (v int64, err error) {
	buf, err := ReadExactly(8, r)
	if err == nil {
		v = int64(binary.BigEndian.Uint64(buf))
	}
	return
}

func BigEndianUint32(val uint32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, val)
	return buf
}

func BigEndianUint64(val int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(val))
	return buf
}

// BoolFromQuery return false if query is missing or set to false
func BoolFromQuery(name string, r *http.Request) bool {
	q := r.URL.Query().Get(name)
	// accepts 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False
	b, _ := strconv.ParseBool(q)
	return b
}

func ToJsonString(data interface{}) string {
	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {return ""}
	return string(jsonData)
}

func Contains(target string, list []string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

func Rewind(r io.ReadSeeker) {
	r.Seek(io.SeekStart,0)
}

func PickUrl(urls []string, preferHttp bool) string {
	pref := "https://"
	if preferHttp {
		pref = "http://"
	}
	for _, u := range urls {
		if strings.HasPrefix(u, pref) {
			return u
		}
	}
	if len(urls) > 0 {
		return urls[0]
	}
	return ""
}

func CRC32(input []byte) []byte {
	computedCrc := crc32.ChecksumIEEE(input)
	return BigEndianUint32(computedCrc)
}

func MakeStringOfLen(length int) string {
	b := make([]byte, length)
	for i, _ := range b {
		b[i] = 'a'
	}
	return string(b)
}