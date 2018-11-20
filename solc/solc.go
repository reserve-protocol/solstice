package solc

import (
    "encoding/json"
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
    AST JSONASTTree
}

type JSONASTTree struct {
    Id int
    Src string
    Children []*JSONASTTree
    // name string
    // attributes, which is a rich collection of information we're not using
}

func GetCombinedJSON(artifactList string, contracts []string) (CombinedJSON, error) {
    var outputJSON CombinedJSON
    solcArgs := append(
        []string{
            "openzeppelin-solidity=./vendor/openzeppelin-solidity",
            "--optimize",
            "--combined-json=" + artifactList,
        },
        contracts...)

    cmd := exec.Command("solc", solcArgs...)
    cmd.Dir = viper.GetString("contracts_dir")
    out, err := cmd.Output()
    if err != nil {
        return CombinedJSON{}, err
    }

    err = json.Unmarshal(out, &outputJSON)
    return outputJSON, err
}
