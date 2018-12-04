package srclocation

import (
    "errors"
    "io"
    "io/ioutil"
    "os"
    "html"
)

// A particular range of bytes in a source file.
type SourceLocation struct {
    ByteOffset     int
    ByteLength     int
    SourceFileName string
    JumpType       rune
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
