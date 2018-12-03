package common

import (
	"bytes"
	"io/ioutil"

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
