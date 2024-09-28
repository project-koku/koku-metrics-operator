package utils

import (
	"os"
	"strconv"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

var logger = ctrl.Log.WithName("miscutils")

// GetEnvVarBool returns the boolean value from an environment variable or the
// provided default boolean value if the variable does not exist.
func GetEnvVarBool(varName string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(varName); exists {
		parsedVal, err := strconv.ParseBool(value)
		if err != nil {
			logger.Error(err, "Invalid boolean format for environment variable", "Variable", varName, "Value", value)
		}
		return parsedVal
	}
	return defaultValue
}

// GetEnvVar returns the value from an environment variable or the
// provided default value if the variable does not exist.
func GetEnvVar(varName, defaultValue string) string {
	if value, exists := os.LookupEnv(varName); exists {
		return value
	}
	return defaultValue
}

// Returns time.Duration parsed from an env variable or a default if
// the variable does not exist or does not parse into a duration.
func GetEnvVarDuration(varName string, defaultValueStr string) time.Duration {
	val := GetEnvVar(varName, "")
	defaultValue, _ := time.ParseDuration(defaultValueStr)

	if val == "" {
		return defaultValue
	}

	if parsedVal, err := time.ParseDuration(val); err == nil {
		return parsedVal

	} else {
		logger.Error(err, "Invalid boolean format for environment variable", "Variable", varName, "Value", val)
		return defaultValue
	}

}
