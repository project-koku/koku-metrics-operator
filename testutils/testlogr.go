//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package testutils

import (
	"fmt"
	"log"
	"os"

	"github.com/go-logr/logr"
)

// TestLogger is a logr.Logger that only prints the main msg.
type TestLogger struct{}

var _ logr.Logger = TestLogger{}
var logger = log.New(os.Stdout, "test logger: ", 64) // log.Lmsgprefix == 64

func (TestLogger) Info(msg string, args ...interface{}) {
	str := ""
	if len(args) > 0 {
		str += fmt.Sprintf(": %v", args)
	}
	logger.Printf("'%s'"+str, msg)
}

func (TestLogger) Enabled() bool {
	return false
}

func (log TestLogger) Error(err error, msg string, args ...interface{}) {
	logger.Printf("'%s': %v -- %v", msg, err, args)
}

func (log TestLogger) V(level int) logr.InfoLogger {
	return log
}

func (log TestLogger) WithName(_ string) logr.Logger {
	return log
}

func (log TestLogger) WithValues(_ ...interface{}) logr.Logger {
	return log
}
