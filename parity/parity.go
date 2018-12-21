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
	JSONRPC string
	Result  execTrace
	ID      int
}

type execTrace struct {
	Trace     []interface{}
	StateDiff interface{}
	Output    string
	VMTrace   VMTrace
}

type VMTrace struct {
	Ops  []traceOperation
	Code string
}

type traceOperation struct {
	Cost int
	PC   int
	Sub  interface{}
	Ex   execTraceOpEx
}

type execTraceOpEx struct {
	Push  []string
	Mem   interface{}
	Used  int
	Store interface{}
}

func GetExecTrace(txnHash string) (VMTrace, error) {
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
		return VMTrace{}, err
	}
	defer resp.Body.Close()

	var execTraceResponse jsonrpcResponse
	err = json.NewDecoder(resp.Body).Decode(&execTraceResponse)
	if err != nil {
		return VMTrace{}, err
	}

	if execTraceResponse.Result.Output == "" {
		return execTraceResponse.Result.VMTrace, errors.New("Transaction ID not found.")
	}

	if execTraceResponse.Result.VMTrace.Code == "0x" {
		return execTraceResponse.Result.VMTrace, errors.New("Transaction has no associated bytecode.")
	}

	if len(execTraceResponse.Result.VMTrace.Ops) == 0 {
		return execTraceResponse.Result.VMTrace, errors.New("Transaction has no execution trace steps.")
	}

	return execTraceResponse.Result.VMTrace, nil
}
