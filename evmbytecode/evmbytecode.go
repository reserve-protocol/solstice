package evmbytecode

import (
	"errors"
	"strings"
)

// This function zeros-out the meta data stored in the bytecode. It is a hash
// of all compilation context, and is impractical to reproduce. Documented here;
// https://solidity.readthedocs.io/en/v0.5.1/metadata.html
func RemoveMetaData(bytecode string) string {
	if len(bytecode) < 2 || !strings.HasPrefix(bytecode, "0x") {
		panic(errors.New("Bytecode must start with 0x."))
	}

	if bytecode == "0x" || len(bytecode) < 18+64+4 {
		return bytecode
	}

	metadataIndex := strings.Index(bytecode, "a165627a7a72305820")

	if metadataIndex == -1 {
		return bytecode
	}

	if bytecode[metadataIndex+18+64:metadataIndex+18+64+4] != "0029" {
		panic(errors.New("Metadata malformed."))
	}

	// If everything looks fine, replace metadata hash with 0's
	return bytecode[0:metadataIndex+18] +
		strings.Repeat("0", 64) +
		bytecode[metadataIndex+18+64:len(bytecode)]
}

// Stands for Get Program Counter--to--Operation Index mapping
func GetPcToOpIndex(bytecode string) map[int]int {
	var pcToOpIndex = make(map[int]int)

	var firstByteRune rune
	var currentByte string
	var byteIndex int

	pushBytesRemaining := 0

	opIndex := 0

	for index, char := range strings.TrimPrefix(bytecode, "0x") {
		if index%2 == 0 {
			firstByteRune = char
			continue
		} else if index%2 == 1 {
			currentByte = string(firstByteRune) + string(char)
			byteIndex = index / 2
		} else {
			panic("This is logically impossible.")
		}

		// Now you have currentByte and byteIndex

		if pushBytesRemaining != 0 {
			pushBytesRemaining -= 1
			continue
		}

		pcToOpIndex[byteIndex] = opIndex

		if pushBytes := bytesPushed(currentByte); pushBytes != -1 {
			pushBytesRemaining = pushBytes
		}

		opIndex += 1
		continue
	}

	return pcToOpIndex
}

func bytesPushed(targetOpCode string) int {
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
			// Plus one because the first PUSH opcode pushes one byte onto the stack, not zero
			return index + 1
		}
	}
	return -1
}
