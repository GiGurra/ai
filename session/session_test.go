package session

import (
	"fmt"
	"testing"
)

func TestHashString(t *testing.T) {
	src1 := "test"
	src2 := "test"
	src3 := "test2"

	hash1 := HashString(src1)
	hash2 := HashString(src2)
	hash3 := HashString(src3)

	if hash1 != hash2 {
		t.Errorf("HashString(%s) = %s; want %s", src1, hash1, hash2)
	}

	if hash1 == hash3 {
		t.Errorf("HashString(%s) = %s; want %s", src1, hash1, hash3)
	}

	fmt.Printf("HashString(%s) = %s\n", src1, hash1)
	fmt.Printf("HashString(%s) = %s\n", src2, hash2)
	fmt.Printf("HashString(%s) = %s\n", src3, hash3)
}

func TestTerminalId(t *testing.T) {
	terminalId1 := TerminalId()
	terminalId2 := TerminalId()
	fmt.Printf("TerminalId() = %s\n", terminalId1)

	if terminalId1 != terminalId2 {
		t.Errorf("TerminalId() = %s; want %s", terminalId1, terminalId2)
	}
}

func TestBootID(t *testing.T) {
	bootID1 := BootID()
	bootID2 := BootID()
	fmt.Printf("BootID() = %s\n", bootID1)

	if bootID1 != bootID2 {
		t.Errorf("BootID() = %s; want %s", bootID1, bootID2)
	}
}
