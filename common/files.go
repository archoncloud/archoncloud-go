package common

import (
	"github.com/dustin/go-humanize"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func GetTempFile() (f *os.File, err error) {
	return ioutil.TempFile("", "archon")
}

// WriteTempFile writes from the reader to a temp file and returns the path of the file
// or an empty string on error
func WriteTempFile(r io.Reader) string {
	tempFile, err := GetTempFile()
	if err != nil {return ""}
	defer tempFile.Close()
	_, err = io.Copy(tempFile, r)
	if err != nil {return ""}
	return tempFile.Name()
}

func IsFullPath( path string ) bool {
	l := len(path)
	if l == 0 {
		return false
	}
	if path[0] == '/' || path[0] == '\\' {
		return true
	}
	if l >= 2 && path[1] == ':' {
		// Windows
		return true
	}
	return false
}

func Join(dir, subpath string) string {
	if IsFullPath(subpath) {return subpath}
	return filepath.Join(dir, subpath)
}

// Default to executable returns full path. If input path is relative, it will considered relative to the
// the folder of the executable
func DefaultToExecutable(path string) string {
	ex, _ := os.Executable()
	return Join(filepath.Dir(ex),path)
}

func CreateFile(fullPath string) (*os.File, error) {
	err := os.MkdirAll(filepath.Dir(fullPath), os.ModeDir|os.ModePerm)
	if err != nil {return nil, err}

	file, err := os.Create(fullPath)
	return file, err
}

func FileExists( path string ) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

func FileSize( path string ) int64 {
	fi, err := os.Stat(path);
	if err != nil {
		return 0
	}
	return fi.Size()
}

func FileSizeString(fullPath string) string {
	return humanize.Bytes(uint64(FileSize(fullPath)))
}

func MakeFolders(folders []string) (err error) {
	for _, folder := range folders {
		err = os.MkdirAll(folder, os.ModeDir|os.ModePerm)
		if err != nil {break}
	}
	return
}


// DirSize returns the size the files in a folder and subfolders in bytes
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}
