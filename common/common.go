package common

import (
	"bytes"
	"errors"
	"io/ioutil"
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

func RemoveMetaData(bytecode string) string {
	if len(bytecode) < 2 || !strings.HasPrefix(bytecode, "0x") {
		panic(errors.New("Bytecode must start with 0x."))
	}

	if bytecode == "0x" || len(bytecode) < 18+64+4 {
		return bytecode
	}

	metadataIndex := strings.Index(bytecode, "a165627a7a72305820")

	if metadataIndex == -1 {
		return bytecode
	}

	if bytecode[metadataIndex+18+64:metadataIndex+18+64+4] != "0029" {
		panic(errors.New("Metadata malformed."))
	}

	// If everything looks fine, replace metadata hash with 0's
	return bytecode[0:metadataIndex+18] +
		strings.Repeat("0", 64) +
		bytecode[metadataIndex+18+64:len(bytecode)]
}
