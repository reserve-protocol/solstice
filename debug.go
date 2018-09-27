// TODO: Things to make configurable; the contract directory, the client port number

package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "go/build"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "strconv"
    "strings"
)

// For parsing the exec trace
// -----------------------------------------------

type jsonrpcResponse struct {
    Jsonrpc string
    Result  execTrace
    Id      int
}

type execTrace struct {
    Trace     []interface{}
    StateDiff interface{}
    Output    string
    VmTrace   vmTrace
}

type vmTrace struct {
    Ops  []traceOperation
    Code string
}

type traceOperation struct {
    Cost int
    Pc   int
    Sub  interface{}
    Ex   execTraceOpEx
}

type execTraceOpEx struct {
    Push  []string
    Mem   interface{}
    Used  int
    Store interface{}
}

// -----------------------------------------------

// For parsing the source map
// -----------------------------------------------
type solcCombinedJson struct {
    Contracts  map[string]srcmapRuntime
    SourceList []string
    Version    string
}

type srcmapRuntime struct {
    SrcmapRuntime string `json:"srcmap-runtime"`
}
// -----------------------------------------------


func main() {
    var txnHash string
    var sourceFilePath string
    var contractName string
    flag.StringVar(&txnHash, "txnHash", "0x0", "a transaction hash")
    flag.StringVar(&sourceFilePath, "sourceFilePath", "", "contract file path")
    flag.StringVar(&contractName, "contractName", "", "contract file name")
    flag.Parse()
    
    if contractName == "" {
        contractName = strings.Split(filepath.Base(sourceFilePath), ".")[0]
    }
    contractsPath := filepath.Join(
        build.Default.GOPATH,
        "/src/github.com/coordination-institute/reserve/protocol/ethereum/contracts",
    )

    var execTraceResponse jsonrpcResponse
    {
        dataString := fmt.Sprintf(
            "{\"jsonrpc\":\"2.0\",\"method\":\"trace_replayTransaction\",\"params\":[\"%s\", [\"vmTrace\"]],\"id\":1}",
            txnHash,
        )
        cmd := exec.Command(
            "curl",
            "-X", "POST", "-H",
            "Content-Type: application/json",
            "--data", dataString,
            "http://127.0.0.1:8545",
        )
        out, err := cmd.Output()
        if err != nil {
            fmt.Println(err)
        }

        err = json.Unmarshal(out, &execTraceResponse)
        if err != nil {
            fmt.Println(err)
        }
    }

    trace := execTraceResponse.Result.VmTrace.Ops
    lastTraceOp := trace[len(trace) - 1]
    lastProgramCounter := lastTraceOp.Pc

    fmt.Printf("Last program counter: %v\n", lastProgramCounter)

    var finalOpIndex int
    {
        bytecode := execTraceResponse.Result.VmTrace.Code[2:] // Remove "0x" from the front
        var firstByteRune rune
        var currentByte string

        pushBytesRemaining := 0

        opIndex := 0

        for index, char := range bytecode {
            if index % 2 == 0 {
                firstByteRune = char
                continue
            } else if index % 2 == 1 {
                currentByte = string(firstByteRune) + string(char)
            } else {
                panic("literally what")
            }

            // Now you have currentByte and index

            if pushBytesRemaining != 0 {
                pushBytesRemaining -= 1
                continue
            }

            if index/2 == lastProgramCounter { // Or something close to this
                finalOpIndex = opIndex
                break
            }

            if pushBytes := indexOf(currentByte); pushBytes != -1 {
                pushBytesRemaining = pushBytes
                // Maybe an off by one error here
            }

            opIndex += 1
            continue
        }

    }
    fmt.Printf("Final op index: %v\n", finalOpIndex)

    // Now you have the finalOpIndex with which to pick an operation from the source map

    var srcMapSlice []string
    var srcList []string
    {
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
            fmt.Println("Getting source map: ", err)
        }
        var srcMapJson solcCombinedJson
        err = json.Unmarshal(out, &srcMapJson)
        if err != nil {
            fmt.Println(err)
        }
        srcMapString := srcMapJson.Contracts[sourceFilePath + ":Auctioneer"].SrcmapRuntime
        srcList = srcMapJson.SourceList
        srcMapSlice = strings.Split(srcMapString, ";")
    }

    var opSourceLocation [4]string

    for i := 0; i < finalOpIndex; i++ {
        instructionTuple := srcMapSlice[i]
        instructionSlice := strings.Split(instructionTuple, ":")
        for j, val := range instructionSlice {
            if val != "" {
                opSourceLocation[j] = val
            }
        }
    }

    fmt.Printf("Instruction tuple %s\n", opSourceLocation)

    byteOffset, err := strconv.Atoi(opSourceLocation[0])
    if err != nil {
        panic(err)
    }

    byteLength, err := strconv.Atoi(opSourceLocation[1])
    if err != nil {
        panic(err)
    }

    sourceFileIndex, err := strconv.Atoi(opSourceLocation[2])
    if err != nil {
        panic(err)
    }

    sourceFileName := filepath.Join(contractsPath, srcList[sourceFileIndex])
    sourceFileReader, err := os.Open(sourceFileName)
    if err != nil {
        panic(err)
    }
    defer sourceFileReader.Close()
    sourceFileBeginning := make([]byte, byteOffset + byteLength)

    _, err = io.ReadFull(sourceFileReader, sourceFileBeginning)
    if err != nil {
        panic(err)
    }

    lineNumber := 1
    columnNumber := 1
    var codeSnippet []byte
    for byteIndex, sourceByte := range sourceFileBeginning {
        if byteIndex < byteOffset {
            columnNumber += 1
            if sourceByte == '\n' {
                lineNumber += 1
                columnNumber = 1
            }
        } else {
            codeSnippet = append(codeSnippet, sourceByte)
        }
    }

    fmt.Printf("%s %d:%d\n", sourceFileName, lineNumber, columnNumber)
    fmt.Printf("... %s ...\n", codeSnippet)
}

func indexOf(targetOpCode string) int {
    var pushOps = [...]string{
        "60",
        "61",
        "62",
        "63",
        "64",
        "65",
        "66",
        "67",
        "68",
        "69",
        "6a",
        "6b",
        "6c",
        "6d",
        "6e",
        "6f",
        "70",
        "71",
        "72",
        "73",
        "74",
        "75",
        "76",
        "77",
        "78",
        "79",
        "7a",
        "7b",
        "7c",
        "7d",
        "7e",
        "7f",
    }

    for index, opCode := range pushOps {
        if opCode == targetOpCode {
            return index + 1 // Plus one because the first PUSH opcode pushes one byte onto the stack, not zero
        }
    }
    return -1
}
