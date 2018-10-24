package trace

import (
	"strings"
)

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
			panic("literally what")
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
			return index + 1 // Plus one because the first PUSH opcode pushes one byte onto the stack, not zero
		}
	}
	return -1
}
