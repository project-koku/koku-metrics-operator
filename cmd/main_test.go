//
// Copyright 2024 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestMain(m *testing.M) {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	os.Exit(m.Run())
}

// Helper function to unset an environment variable
func unsetEnvVar(t *testing.T, key string) func() {
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Failed to unset env var %s: %v", key, err)
	}
	return func() {}
}

// Helper function to set an environment variable
func setEnvVar(t *testing.T, key, value string) func() {
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Failed to unset env var %s: %v", key, value)
	}

	return func() {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("Failed to unset env var %s: %v", key, err)
		}
	}
}

// Test getWatchNamespace function
func TestGetWatchNamespace(t *testing.T) {
	const watchNamespaceEnvVar = "WATCH_NAMESPACE"

	testCases := []struct {
		name           string
		envVarSet      bool
		envValue       string
		expectedResult string
		expectError    bool
	}{
		{
			name:           "Env variable is set",
			envVarSet:      true,
			envValue:       "test-namespace",
			expectedResult: "test-namespace",
			expectError:    false,
		},
		{
			name:        "Env variable is not set",
			envVarSet:   false,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envVarSet {
				defer setEnvVar(t, watchNamespaceEnvVar, tc.envValue)()
			} else {
				defer unsetEnvVar(t, watchNamespaceEnvVar)()
			}

			ns, err := getWatchNamespace()
			if tc.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else {
					expectedErr := fmt.Sprintf("%s must be set", watchNamespaceEnvVar)
					if err.Error() != expectedErr {
						t.Errorf("expected error '%s', got '%s'", expectedErr, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if ns != tc.expectedResult {
					t.Errorf("expected namespace '%s', got '%s'", tc.expectedResult, ns)
				}
			}
		})
	}
}

// Test getEnvVarString function
func TestGetEnvVarString(t *testing.T) {
	const varName = "TEST_ENV_VAR"
	const defaultValue = "default"

	testCases := []struct {
		name          string
		envVarSet     bool
		envValue      string
		expectedValue string
	}{
		{
			name:          "Env variable is set",
			envVarSet:     true,
			envValue:      "test value",
			expectedValue: "test value",
		},
		{
			name:          "Env variable is not set",
			envVarSet:     false,
			expectedValue: defaultValue,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envVarSet {
				defer setEnvVar(t, varName, tc.envValue)()
			} else {
				defer unsetEnvVar(t, varName)()
			}

			value := getEnvVarString(varName, defaultValue)
			if value != tc.expectedValue {
				t.Errorf("expected value '%s', got '%s'", tc.expectedValue, value)
			}
		})
	}
}

// Test getEnvVarDuration function
func TestGetEnvVarDuration(t *testing.T) {
	const varName = "TEST_DURATION_ENV_VAR"
	const defaultValue = 10 * time.Second

	testCases := []struct {
		name          string
		envVarSet     bool
		envValue      string
		expectedValue time.Duration
	}{
		{
			name:          "Env variable is set to valid duration",
			envVarSet:     true,
			envValue:      "15s",
			expectedValue: 15 * time.Second,
		},
		{
			name:          "Env variable is not set",
			envVarSet:     false,
			expectedValue: defaultValue,
		},
		{
			name:          "Env variable is set to invalid duration",
			envVarSet:     true,
			envValue:      "invalid-duration",
			expectedValue: defaultValue,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envVarSet {
				defer setEnvVar(t, varName, tc.envValue)()
			} else {
				defer unsetEnvVar(t, varName)()
			}

			value := getEnvVarDuration(varName, defaultValue)
			if value != tc.expectedValue {
				t.Errorf("expected duration '%v', got '%v'", tc.expectedValue, value)
			}
		})
	}
}

// Test validateLeaderElectionConfig function
func TestValidateLeaderElectionConfig(t *testing.T) {
	testCases := []struct {
		name                  string
		leaseDuration         time.Duration
		renewDeadline         time.Duration
		retryPeriod           time.Duration
		expectedLeaseDuration time.Duration
		expectedRenewDeadline time.Duration
		expectedRetryPeriod   time.Duration
	}{
		{
			name:                  "All durations valid",
			leaseDuration:         60 * time.Second,
			renewDeadline:         30 * time.Second,
			retryPeriod:           5 * time.Second,
			expectedLeaseDuration: 60 * time.Second,
			expectedRenewDeadline: 30 * time.Second,
			expectedRetryPeriod:   5 * time.Second,
		},
		{
			name:                  "renewDeadline >= leaseDuration",
			leaseDuration:         30 * time.Second,
			renewDeadline:         60 * time.Second,
			retryPeriod:           5 * time.Second,
			expectedLeaseDuration: 60 * time.Second,
			expectedRenewDeadline: 30 * time.Second,
			expectedRetryPeriod:   5 * time.Second,
		},
		{
			name:                  "retryPeriod >= renewDeadline",
			leaseDuration:         60 * time.Second,
			renewDeadline:         30 * time.Second,
			retryPeriod:           30 * time.Second,
			expectedLeaseDuration: 60 * time.Second,
			expectedRenewDeadline: 30 * time.Second,
			expectedRetryPeriod:   5 * time.Second,
		},
		{
			name:                  "Both conditions invalid",
			leaseDuration:         30 * time.Second,
			renewDeadline:         60 * time.Second,
			retryPeriod:           60 * time.Second,
			expectedLeaseDuration: 60 * time.Second,
			expectedRenewDeadline: 30 * time.Second,
			expectedRetryPeriod:   5 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ld, rd, rp := validateLeaderElectionConfig(tc.leaseDuration, tc.renewDeadline, tc.retryPeriod)
			if ld != tc.expectedLeaseDuration || rd != tc.expectedRenewDeadline || rp != tc.expectedRetryPeriod {
				t.Errorf("expected ld=%v, rd=%v, rp=%v; got ld=%v, rd=%v, rp=%v",
					tc.expectedLeaseDuration, tc.expectedRenewDeadline, tc.expectedRetryPeriod, ld, rd, rp)
			}
		})
	}
}
