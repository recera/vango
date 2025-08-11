//go:build js && wasm
// +build js,wasm

package wasm

import (
	"strconv"
)

// Sprintf is a simple sprintf implementation for WASM that avoids fmt package
func Sprintf(format string, args ...interface{}) string {
	// This is a very basic implementation
	// In production, we'd want something more robust
	result := format
	
	for i, arg := range args {
		placeholder := "%" + strconv.Itoa(i)
		
		switch v := arg.(type) {
		case string:
			result = stringReplace(result, placeholder, v)
		case int:
			result = stringReplace(result, placeholder, strconv.Itoa(v))
		case uint32:
			result = stringReplace(result, placeholder, strconv.FormatUint(uint64(v), 10))
		case bool:
			result = stringReplace(result, placeholder, strconv.FormatBool(v))
		default:
			// For other types, just use a placeholder
			result = stringReplace(result, placeholder, "[value]")
		}
	}
	
	return result
}

// Simple string replace function
func stringReplace(s, old, new string) string {
	// This is inefficient but works for our needs
	for {
		idx := stringIndex(s, old)
		if idx == -1 {
			break
		}
		s = s[:idx] + new + s[idx+len(old):]
	}
	return s
}

// Find index of substring
func stringIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}