package common

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func GetFromSP(spUrl, endPoint, query string, timeout time.Duration) (contents string, err error) {
	getUrl := fmt.Sprintf("%s/%s", strings.TrimRight(spUrl, "/"), strings.TrimLeft(endPoint, "/"))
	if query != "" {
		getUrl += "?" + query
	}
	spConn := http.Client{Timeout: timeout}
	response, err := spConn.Get(getUrl); if err != nil {return}
	defer response.Body.Close()
	return GetResponse(response)
}

// GetResponse is read the http response body and returns text or error
func GetResponse(resp *http.Response) (string, error) {
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {return "", err}

	respText := string(respBody)
	if resp.StatusCode != http.StatusOK {
		if respText != "" {
			// Return the response text as it is more descriptive of the problem
			err = errors.New(respText)
		} else {
			err = errors.New(resp.Status)
		}
		return "", err
	}
	return respText, nil
}

