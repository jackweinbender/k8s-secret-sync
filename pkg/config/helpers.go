package config

import (
	"os"
	"strconv"
)

// envVar is a type constraint that matches string, int, and bool types.
type envVar interface {
	string | int | bool
}

// env returns the value of the environment variable named by envVar,
// or defaultValue if the environment variable is not present or cannot be parsed.
// The type of the return value matches the type of defaultValue.
func env[T envVar](envVar string, defaultValue T) T {
	if value := os.Getenv(envVar); value != "" {
		switch any(defaultValue).(type) {
		case string:
			return any(value).(T)
		case int:
			intValue, err := strconv.Atoi(value)
			if err == nil {
				return any(intValue).(T)
			}
		case bool:
			boolValue, err := strconv.ParseBool(value)
			if err == nil {
				return any(boolValue).(T)
			}
		}
	}
	return defaultValue
}
