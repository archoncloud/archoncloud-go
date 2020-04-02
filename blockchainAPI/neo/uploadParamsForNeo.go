package neo

import (
	"fmt"
	. "github.com/archoncloud/archoncloud-go/common"
	"strconv"
	"strings"
)

type UploadParamsForNeo struct {
	UserName			string
	PublicKey          string
	ContainerSignature string
	FileContainerType  int
	SPsToUploadTo      []string // addresses of sps to whom the shards are to go
}

func (u *UploadParamsForNeo) String() string {
	s := SeparatedStringList(stringSep,
		u.UserName,
		u.PublicKey,
		u.ContainerSignature,
		u.FileContainerType,
		u.SPsToUploadTo)
	return s
}

func (u *UploadParamsForNeo) Bytes() []byte {
	s := u.String()
	return CompressString(s)
}

func NewUploadParamsForNeo(s string) (u *UploadParamsForNeo, err error) {
	buf := strings.Split(s, stringSep)
	if len(buf) < 5 {
		err = fmt.Errorf("invalid Neo upload string: %q", s)
		return
	}
	u = new(UploadParamsForNeo)
	u.UserName = buf[0]
	u.PublicKey = buf[1]
	u.ContainerSignature = buf[2]
	u.FileContainerType, _ = strconv.Atoi(buf[3])
	for _, spa := range buf[4:] {
		u.SPsToUploadTo = append(u.SPsToUploadTo,spa)
	}
	return
}

func NewUploadParamsForNeoFromBytes(b []byte) (u *UploadParamsForNeo, err error) {
	s := UnCompressString(b)
	return NewUploadParamsForNeo(s)
}