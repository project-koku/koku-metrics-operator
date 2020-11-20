/*


Copyright 2020 Red Hat, Inc.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package testlogr

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
