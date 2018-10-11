package parity

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type jsonrpcResponse struct {
	Jsonrpc string
	Result  execTrace
	Id      int
}

type execTrace struct {
	Trace     []interface{}
	StateDiff interface{}
	Output    string
	VmTrace   VmTrace
}

type VmTrace struct {
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

func GetExecTrace(txnHash string) (VmTrace, error) {
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

	var execTraceResponse jsonrpcResponse

	out, err := cmd.Output()
	if err != nil {
		return execTraceResponse.Result.VmTrace, err
	}

	err = json.Unmarshal(out, &execTraceResponse)
	if err != nil {
		return execTraceResponse.Result.VmTrace, err
	}

	return execTraceResponse.Result.VmTrace, err
}
