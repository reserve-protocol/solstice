package srcmap

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/coordination-institute/debugging-tools/common"
)

type solcCombinedJSON struct {
	Contracts  map[string]runtimeArtifacts
	SourceList []string
	Version    string
}

type runtimeArtifacts struct {
	SrcmapRuntime string `json:"srcmap-runtime"`
	BinRuntime    string `json:"bin-runtime"`
}

func Get(contractsPath string) (map[string][]common.OpSourceLocation, map[string]string, error) {
	var srcMapJSON solcCombinedJSON

	// TODO: Make this list .sol files recursively
	files, err := filepath.Glob(contractsPath + "/*.sol")
	if err != nil {
		return map[string][]common.OpSourceLocation{}, map[string]string{}, err
	}

	solcArgs := append(
		[]string{
			"openzeppelin-solidity=./vendor/openzeppelin-solidity",
			"rocketpool=./vendor/rocketpool",
			"--optimize",
			"--combined-json=srcmap-runtime,bin-runtime",
		},
		files...)

	cmd := exec.Command("solc", solcArgs...)
	cmd.Dir = contractsPath
	out, err := cmd.Output()
	if err != nil {
		return map[string][]common.OpSourceLocation{}, map[string]string{}, err
	}

	err = json.Unmarshal(out, &srcMapJSON)
	if err != nil {
		return map[string][]common.OpSourceLocation{}, map[string]string{}, err
	}

	bytecodeToFilename := make(map[string]string)
	for contractName, artifacts := range srcMapJSON.Contracts {
		if len(artifacts.BinRuntime) != 0 {
			bytecode := "0x" + artifacts.BinRuntime
			// TODO: Make "removeMetaData" function that asserts that the
			// metadata is there, and then cuts it off.
			bytecodeToFilename[common.RemoveMetaData(bytecode)] = contractName
		}
	}

	sourceMaps := map[string][]common.OpSourceLocation{}
	for contractName, artifacts := range srcMapJSON.Contracts {
		srcMapSlice := strings.Split(artifacts.SrcmapRuntime, ";")

		var opSourceLocations []common.OpSourceLocation
		for i, instructionTuple := range srcMapSlice {
			var currentStruct common.OpSourceLocation
			if i == 0 {
				currentStruct = common.OpSourceLocation{}
			} else {
				currentStruct = opSourceLocations[len(opSourceLocations)-1]
			}
			for j, val := range strings.Split(instructionTuple, ":") {
				// We do this because the split tuple might have any length <= 4.
				// Most of these cases won't be hit for most tuples.
				if val != "" {
					var err error
					if j == 0 {
						currentStruct.ByteOffset, err = strconv.Atoi(val)
					} else if j == 1 {
						currentStruct.ByteLength, err = strconv.Atoi(val)
					} else if j == 2 {
						sourceFileIndex, err := strconv.Atoi(val)
						if err != nil {
							return sourceMaps, bytecodeToFilename, err
						}
						if sourceFileIndex != -1 {
							currentStruct.SourceFileName = srcMapJSON.SourceList[sourceFileIndex]
						} else {
							currentStruct.SourceFileName = ""
						}
					} else if j == 3 {
						currentStruct.JumpType = rune(val[0])
					}
					if err != nil {
						return sourceMaps, bytecodeToFilename, err
					}
				}
			}
			opSourceLocations = append(opSourceLocations, currentStruct)
		}
		sourceMaps[contractName] = opSourceLocations
	}
	return sourceMaps, bytecodeToFilename, err
}
