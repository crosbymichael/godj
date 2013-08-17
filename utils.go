package godj

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
)

func debug(format string, v ...interface{}) {
	format = "[godj]: " + format
	log.Printf(format, v...)
}

func readHint(j *Journal) (int, error) {
	hint, err := os.Open(path.Join(j.FullPath(), HINT))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	raw, err := ioutil.ReadAll(hint)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(raw))
}

func writeHint(j *Journal) error {
	length := []byte(strconv.Itoa(j.hint))
	p := path.Join(j.FullPath(), HINT)
	return ioutil.WriteFile(p, length, JOURNALPERMISSION)
}
