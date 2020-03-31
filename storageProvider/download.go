package storageProvider

import (
	"fmt"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/pkg/errors"
	"net/http"
	"path"
	"strconv"
)

/* This is now obsolete
// To download a shard indirectly
type shardsDir string
type hashesDir string

// Open implements FileSystem interface for /arc endpoint (for download)
func (sd shardsDir) Open(name string) (http.File, error) {
	LogTrace.Printf("Download request on /arc for: %q\n", name)
	hashPath, err := storedFilePath(name)
	if err != nil {
		return nil, err
	}
	return downloadShard(fp.Base(hashPath))
}

// Open implements FileSystem interface for /hash endpoint (for download)
func (hd hashesDir) Open(name string) (http.File, error) {
	if !strings.HasSuffix(name, HashFileSuffix) {
		return nil, fmt.Errorf("file needs a %q extension", HashFileSuffix)
	}
	return downloadShard(name)
}

// downloadShard returns the file to be downloaded. http.FileServer does the rest
func downloadShard(name string) (http.File, error) {
	file, err := http.Dir(GetHashesFolder()).Open(name)
	if err != nil {
		LogError.Println( err )
		return nil, err
	}
	LogInfo.Printf("Shard downloaded: %q\n", name)
	return file, nil
}

*/

// downloadHandler implements the /download endpoint
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	// /download?shardIdx=X&archonUrl=arc://george.n2:7/photos/vacation/Miami/beach.jpg
	var err error
	shardIx := -1
	aUrl, err := NewArchonUrl(r.URL.Query().Get(ArchonUrlQuery))
	if err == nil {
		if !aUrl.IsWholeFile() {
			shardIxS := r.URL.Query().Get(ShardIdxQuery)
			if shardIxS == "" {
				err = errors.New("missing " + ShardIdxQuery)
			} else {
				ix, err1 := strconv.ParseInt(shardIxS, 10, 8)
				shardIx = int(ix)
				err = err1
			}
		}
	}
	if err != nil {
		httpBadRequest(w, r, err)
		return
	}

	shardPath := aUrl.ShardPath(int(shardIx))
	LogTrace.Printf("Download request on /download for: %q\n", shardPath)
	var storedPath string
	if aUrl.IsHash() {
		storedPath = InHashesFolder(shardPath)
	} else {
		storedPath = InShardsFolder(shardPath)
	}
	if !FileExists(storedPath) {
		msg := fmt.Sprintf("file %s does not exist", shardPath)
		LogWarning.Println(msg)
		httpBadRequest(w, r, errors.New(msg))
		return
	}

	_, name := path.Split(shardPath)
	a := ActivityInfo{
		Upload:   false,
		Url:      aUrl,
		Client:   r.RemoteAddr,
		NumBytes: FileSize(storedPath),
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s",name))
	http.ServeFile(w, r, storedPath)
	RecordActivity(&a)
}
