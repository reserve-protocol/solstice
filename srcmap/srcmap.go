package srcmap

import (
	"errors"
	"html"
	"io"
	"os"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

    "github.com/spf13/viper"

	"github.com/coordination-institute/debugging-tools/evmbytecode"
	"github.com/coordination-institute/debugging-tools/solc"
)

type SourceLocation struct {
	ByteOffset     int
	ByteLength     int
	SourceFileName string
	JumpType       rune
}

func Get() (map[string][]SourceLocation, map[string]string, error) {
	var files []string
	err := filepath.Walk(viper.GetString("contracts_dir"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".sol") {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return map[string][]SourceLocation{}, map[string]string{}, err
	}

	srcMapJSON, err := solc.GetCombinedJSON("srcmap-runtime,bin-runtime", files)
	if err != nil {
		return map[string][]SourceLocation{}, map[string]string{}, err
	}

	bytecodeToFilename := make(map[string]string)
	for contractName, artifacts := range srcMapJSON.Contracts {
		if len(artifacts.BinRuntime) != 0 {
			bytecode := "0x" + artifacts.BinRuntime
			bytecodeToFilename[evmbytecode.RemoveMetaData(bytecode)] = contractName
		}
	}

	sourceMaps := map[string][]SourceLocation{}
	for contractName, artifacts := range srcMapJSON.Contracts {
		srcMapSlice := strings.Split(artifacts.SrcmapRuntime, ";")

		var opSourceLocations []SourceLocation
		for i, instructionTuple := range srcMapSlice {
			var currentStruct SourceLocation
			if i == 0 {
				currentStruct = SourceLocation{}
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


type ASTTree struct {
	Id uint
	SrcLoc SourceLocation
	Children []*ASTTree
}

func GetAST(contractName string) (ASTTree, error) {
	srcMapJSON, err := solc.GetCombinedJSON("ast", []string{contractName})
	if err != nil {
		return ASTTree{}, err
	}

	processedTree, err := processASTNode(
		srcMapJSON.Sources[contractName].AST,
		srcMapJSON.SourceList,
	)

	return processedTree, err
}

func processASTNode(node solc.JSONASTTree, sourceList []string) (ASTTree, error) {
	var newTree ASTTree
	newTree.Id = node.Id

	srcLocParts := strings.Split(node.Src, ":")

	byteOffset, err := strconv.Atoi(srcLocParts[0])
	if err != nil {
		return newTree, err
	}

	byteLength, err := strconv.Atoi(srcLocParts[1])
	if err != nil {
		return newTree, err
	}

	sourceFileIndex, err := strconv.Atoi(srcLocParts[2])
	if err != nil {
		return newTree, err
	}

	newTree.SrcLoc = SourceLocation{
		byteOffset,
		byteLength,
		sourceList[sourceFileIndex],
		*new(rune),
	}

	for _, childNode := range node.Children {
		newNode, err := processASTNode(*childNode, sourceList)
		if err != nil {
			return newTree, err
		}
		newTree.Children = append(newTree.Children, &newNode)
	}

	return newTree, nil
}


// ~~~~~~~ Sorting ASTTree nodes ~~~~~~~
type byByteOffset []*ASTTree

func (s byByteOffset) Len() int {
    return len(s)
}

func (s byByteOffset) Swap(i, j int) {
    s[i], s[j] = s[j], s[i]
}

func (s byByteOffset) Less(i, j int) bool {
	// If they have the same byte offset, my current belief is that that can
	// only happen if one of them is empty. In that case, we're going to throw
	// it away anyway.
    return s[i].SrcLoc.ByteOffset < s[j].SrcLoc.ByteOffset
}
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type CoverageLoc struct {
	HitCount      int
	CoverageRange SourceLocation
	SrcLocs       []SourceLocation
}

func GetCoverageLocs(filename string) ([]CoverageLoc, error) {
	plainAST, err := GetAST(filename)
	if err != nil {
		return []CoverageLoc{}, err
	}

	covLocs, err := ToCoverageLocs(plainAST)
	if err != nil {
		return []CoverageLoc{}, err
	}
	return covLocs, nil
	// sort coverageLocs by .CoverageRange.ByteOffset
}

func ToCoverageLocs(node ASTTree) ([]CoverageLoc, error) {
	var covLocs []CoverageLoc
	var covLoc CoverageLoc
	covLoc.CoverageRange = node.SrcLoc

	covLoc.SrcLocs = append(covLoc.SrcLocs, node.SrcLoc)
	children := byByteOffset(node.Children)
	sort.Sort(children)
	for _, childNode := range children {
		if childNode.SrcLoc.ByteLength == 0 {
			// This node has no representation in the source file and neither could its children.
			continue
		} else if node.SrcLoc.ByteLength == childNode.SrcLoc.ByteLength {
			// This node is equal in size to its parent. Any siblings must be empty.
			// Not sure what to do here yet, let's roll with it and see what happens.
		}

		rightSrcLoc := covLoc.SrcLocs[len(covLoc.SrcLocs)-1]

		byteOffset1 := rightSrcLoc.ByteOffset
		byteLength1 := childNode.SrcLoc.ByteOffset - rightSrcLoc.ByteOffset
		if byteLength1 < 0 {
			return covLocs, errors.New("Negative byte length1")
		}
		byteOffset2 := childNode.SrcLoc.ByteOffset + childNode.SrcLoc.ByteLength
		byteLength2 := rightSrcLoc.ByteOffset + rightSrcLoc.ByteLength - byteOffset2
		if byteLength2 < 0 {
			return covLocs, errors.New("Negative byte length2")
		}

		covLoc.SrcLocs = append(
			// Cut off the last one, since we just split it in two.
			covLoc.SrcLocs[:len(covLoc.SrcLocs)-1],
			SourceLocation{
				byteOffset1,
				byteLength1,
				node.SrcLoc.SourceFileName,
				*new(rune),
			},
			SourceLocation{
				byteOffset2,
				byteLength2,
				node.SrcLoc.SourceFileName,
				*new(rune),
			},
		)

		childCovLocs, err := ToCoverageLocs(*childNode)
		if err == nil {
			covLocs = append(covLocs, childCovLocs...)
		}

	}
	covLocs = append(covLocs, covLoc)
	// TODO: sort coverLocs
	return covLocs, nil
}




func (location SourceLocation) ByteLocToSnippet() (int, int, []byte, error) {
	sourceFileReader, err := os.Open(location.SourceFileName)
	if err != nil {
		return 0, 0, []byte{}, err
	}
	sourceFileBeginning := make([]byte, location.ByteOffset+location.ByteLength)

	_, err = io.ReadFull(sourceFileReader, sourceFileBeginning)
	if err != nil {
		return 0, 0, []byte{}, err
	}
	defer sourceFileReader.Close()

	lineNumber := 1
	columnNumber := 1
	var codeSnippet []byte
	for byteIndex, sourceByte := range sourceFileBeginning {
		if byteIndex < location.ByteOffset {
			columnNumber += 1
			if sourceByte == '\n' {
				lineNumber += 1
				columnNumber = 1
			}
		} else {
			codeSnippet = append(codeSnippet, sourceByte)
		}
	}
	return lineNumber, columnNumber, codeSnippet, nil
}

const GithubGreen string = "#e6ffed"
const GithubRed string = "#ffeef0"

func (location SourceLocation) LocationMarkup() ([]byte, error) {
	if location.SourceFileName == "" {
		return []byte{}, errors.New("Step source file not found.")
	}

	wholeSrc, err := ioutil.ReadFile(location.SourceFileName)
	if err != nil {
		return []byte{}, err
	}

	srcBeginning := html.EscapeString(string(wholeSrc[0:location.ByteOffset]))
	srcMiddle := html.EscapeString(string(wholeSrc[location.ByteOffset : location.ByteOffset+location.ByteLength]))
	srcEnd := html.EscapeString(string(wholeSrc[location.ByteOffset+location.ByteLength:]))

	return []byte("<pre>" +
		srcBeginning +
		"<span style=\"background-color:" + GithubGreen + ";\">" +
		srcMiddle +
		"</span>" +
		srcEnd +
		"</pre>",
	), nil
}
