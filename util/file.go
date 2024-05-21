package util

import (
	"encoding/json"
	"fmt"
	"os"
)

func ReadFaileAsJson[T any](path string) (T, error) {
	var zero T

	bytes, err := os.ReadFile(path)
	if err != nil {
		return zero, fmt.Errorf("failed to read file: %w", err)
	}

	var result T
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return zero, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	return result, nil
}

func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, fmt.Errorf("failed to stat file: %w", err)
		}
	}
	return true, nil
}

func Must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}
