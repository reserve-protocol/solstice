package source_map

import(
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
)

type solcCombinedJson struct {
	Contracts  map[string]srcmapRuntime
	SourceList []string
	Version    string
}

type srcmapRuntime struct {
	SrcmapRuntime string `json:"srcmap-runtime"`
}

type opSourceLocation struct {
	ByteOffset int
	ByteLength int
	SourceFileIndex int
	JumpType rune
}

func GetSourceMap(sourceFilePath string, contractsPath string) ([]opSourceLocation, []string, error) {
	var opSourceLocations []opSourceLocation
	var srcMapJson solcCombinedJson

	cmd := exec.Command(
		"solc",
		"openzeppelin-solidity=./vendor/openzeppelin-solidity",
		"rocketpool=./vendor/rocketpool",
		"--optimize",
		"--combined-json=srcmap-runtime",
		sourceFilePath,
	)
	cmd.Dir = contractsPath
	out, err := cmd.Output()
	if err != nil {
		return opSourceLocations, srcMapJson.SourceList, err
	}
	err = json.Unmarshal(out, &srcMapJson)
	if err != nil {
		return opSourceLocations, srcMapJson.SourceList, err
	}
	srcMapString := srcMapJson.Contracts[sourceFilePath+":Auctioneer"].SrcmapRuntime
	srcMapSlice := strings.Split(srcMapString, ";")

	for i, instructionTuple := range srcMapSlice {
		var currentStruct opSourceLocation
		if i == 0 {
			currentStruct = opSourceLocation{}
		} else {
			currentStruct = opSourceLocations[len(opSourceLocations) - 1]
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
					currentStruct.SourceFileIndex, err = strconv.Atoi(val)
				} else if j == 3 {
					currentStruct.JumpType = rune(val[0])
				}
				if err != nil {
					return opSourceLocations, srcMapJson.SourceList, err
				}
			}
		}
		opSourceLocations = append(opSourceLocations, currentStruct)
	}

	return opSourceLocations, srcMapJson.SourceList, err
}
