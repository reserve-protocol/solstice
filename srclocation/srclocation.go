package srclocation

import (
	"errors"
	"html"
	"io"
	"io/ioutil"
	"os"
)

// A particular range of bytes in a source file.
type SourceLocation struct {
	ByteOffset     int
	ByteLength     int
	SourceFileName string
	JumpType       rune
}

// It can be the case that srcloc1.Overlaps(srcloc2) AND srcloc2.Overlaps(srcloc1),
// if they're the same size.
func (location1 SourceLocation) Overlaps(location2 SourceLocation) bool {
	startsBefore := location1.ByteOffset <= location2.ByteOffset
	extendsFurther := location2.ByteOffset + location2.ByteLength <= location1.ByteOffset + location1.ByteLength
	if startsBefore && extendsFurther {
		return true
	}
	return false
}

func (location1 SourceLocation) IsDisjointWith(location2 SourceLocation) bool {
	isLeftOf := location1.ByteOffset + location1.ByteLength <= location2.ByteOffset
	isRightOf := location2.ByteOffset + location2.ByteLength <= location1.ByteOffset
	if isLeftOf || isRightOf {
		return true
	}
	return false
}

func (location1 SourceLocation) PartiallyOverlaps(location2 SourceLocation) bool {
	if location1.Overlaps(location2) || location1.IsDisjointWith(location2) {
		return false
	}
	return true
}

func (location SourceLocation) ByteLocToSnippet() (int, int, []byte, error) {
	sourceFileReader, err := os.Open(location.SourceFileName)
	if err != nil {
		return 0, 0, nil, err
	}
	sourceFileBeginning := make([]byte, location.ByteOffset+location.ByteLength)

	_, err = io.ReadFull(sourceFileReader, sourceFileBeginning)
	if err != nil {
		return 0, 0, nil, err
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
		return nil, errors.New("Step source file not found.")
	}

	wholeSrc, err := ioutil.ReadFile(location.SourceFileName)
	if err != nil {
		return nil, err
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
