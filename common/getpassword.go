package common

import (
	"bufio"
	"fmt"
	"github.com/howeyc/gopass"
	"os"
	"strings"
)

func GetPassword(prompt string, show bool) (password string) {
	fmt.Print(prompt + " password: ")
	if show {
		reader := bufio.NewReader(os.Stdin)
		text, err := reader.ReadString('\n')
		Abort(err)
		password = strings.TrimSuffix(text,"\n")
	} else {
		pwd, _ := gopass.GetPasswdMasked()
		password = string(pwd)
	}
	return
}
