package common

import (
	"bytes"
	"io/ioutil"
    "os"
    "path/filepath"
    "strings"

    "github.com/spf13/viper"
)

func Check(err error) {
	if err != nil {
		panic(err)
	}
}

func ReadConfig() error {
    viper.SetConfigName("sample_config")
    viper.AddConfigPath("/home/altair/go/src/github.com/coordination-institute/debugging-tools")
    return viper.ReadInConfig()
}

func NumberOfLines(filename string) (int, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return 0, err
	}
	return bytes.Count(b, []byte{'\n'}), nil
}

func AllContracts() ([]string, error) {
    var filenames []string
    err := filepath.Walk(viper.GetString("contracts_dir"), func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if !info.IsDir() && strings.HasSuffix(info.Name(), ".sol") {
            filenames = append(filenames, path)
        }
        return nil
    })
    return filenames, err
}
