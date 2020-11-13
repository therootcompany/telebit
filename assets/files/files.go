package files

import (
	"net/http"
	"os"
)

func Open(pathstr string) (http.File, error) {
	f, err := Assets.Open(pathstr)
	if nil != err {
		f, err = os.Open(pathstr)
		if nil != err {
			return nil, err
		}
	}
	return f, nil
}
