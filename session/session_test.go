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
