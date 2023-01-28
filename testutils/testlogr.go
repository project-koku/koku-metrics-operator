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

var _ logr.LogSink = TestLogger{}
var logger = log.New(os.Stdout, "test logger: ", 64) // log.Lmsgprefix == 64

func (TestLogger) Init(logr.RuntimeInfo) {
}

func (TestLogger) Info(level int, msg string, args ...interface{}) {
	str := ""
	if len(args) > 0 {
		str += fmt.Sprintf(": %v", args)
	}
	logger.Printf("'%s'"+str, msg)
}

func (TestLogger) Enabled(level int) bool {
	return false
}

func (log TestLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	logger.Printf("'%s': %v -- %v", msg, err, keysAndValues)
}

func (log TestLogger) WithName(name string) logr.LogSink {
	return log
}

func (log TestLogger) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return log
}
