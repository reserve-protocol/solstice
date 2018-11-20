package parity

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

    "github.com/spf13/viper"
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
	resp, err := http.Post(
		viper.GetString("blockchain_client"),
		"application/json",
		strings.NewReader(
			fmt.Sprintf(
				`{
					"jsonrpc": "2.0",
					"method": "trace_replayTransaction",
					"params": [
						%q,
						["vmTrace"]
					],
					"id": 1
				}`,
				txnHash,
			),
		),
	)
	if err != nil {
		return VmTrace{}, err
	}
	defer resp.Body.Close()

	var execTraceResponse jsonrpcResponse
	err = json.NewDecoder(resp.Body).Decode(&execTraceResponse)
	if err != nil {
		return VmTrace{}, err
	}

	if execTraceResponse.Result.Output == "" {
		return execTraceResponse.Result.VmTrace, errors.New("Transaction ID not found.")
	}

	if execTraceResponse.Result.VmTrace.Code == "0x" {
		return execTraceResponse.Result.VmTrace, errors.New("Transaction has no associated bytecode.")
	}

	if len(execTraceResponse.Result.VmTrace.Ops) == 0 {
		return execTraceResponse.Result.VmTrace, errors.New("Transaction has no execution trace steps.")
	}

	return execTraceResponse.Result.VmTrace, nil
}
