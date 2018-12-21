package solc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/spf13/viper"
)

type CombinedJSON struct {
	Contracts  map[string]runtimeArtifacts
	SourceList []string
	Sources    map[string]topASTNode
}

type runtimeArtifacts struct {
	SrcmapRuntime string `json:"srcmap-runtime"`
	BinRuntime    string `json:"bin-runtime"`
}

type topASTNode struct {
	AST JSONAST
}

type JSONAST struct {
	ID       uint
	Src      string
	Children []*JSONAST
	// name string
	// attributes, which is a rich collection of information we're not using
}

func GetCombinedJSON(artifactList string, contracts []string) (CombinedJSON, error) {
	var outputJSON CombinedJSON
	solcArgs := append(
		append(
			viper.GetStringSlice("solc_args"),
			"--combined-json="+artifactList,
		),
		contracts...,
	)
	cmd := exec.Command("solc", solcArgs...)
	cmd.Dir = viper.GetString("contracts_dir")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return CombinedJSON{}, err
	}

	err = json.Unmarshal(out.Bytes(), &outputJSON)
	return outputJSON, err
}
