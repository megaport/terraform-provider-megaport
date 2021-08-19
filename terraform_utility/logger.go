// Copyright 2020 Megaport Pty Ltd
//
// Licensed under the Mozilla Public License, Version 2.0 (the
// "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//       https://mozilla.org/MPL/2.0/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package terraform_utility

import (
	"fmt"
	"log"
	"strings"
)

type Level int8

const (
	TraceLevel Level = iota
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
	Off
)

func (l Level) String() string {
	switch l {
	case TraceLevel:
		return "trace"
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case Off:
		return "off"
	default:
		return "unknown"
	}
}

func StringToLogLevel(level string) Level {
	switch level {
	case "TRACE":
		return TraceLevel
	case "DEBUG":
		return DebugLevel
	case "INFO":
		return InfoLevel
	case "WARN":
		return WarnLevel
	case "ERROR":
		return ErrorLevel
	default:
		return Off
	}
}

type MegaportLogger struct {
	level Level
}

func NewMegaportLogger() *MegaportLogger {
	d := MegaportLogger{level: DebugLevel}
	return &d
}

func (d *MegaportLogger) SetLevel(l Level) {
	d.level = l
}

func (d *MegaportLogger) log(level Level, args ...interface{}) {
	if level >= d.level {
		msg := fmt.Sprint(args...)
		log.Println(fmt.Sprintf("[%s] %s", strings.ToUpper(level.String()), msg))
	}
}

// Emit the message and args at DEBUG level
func (d *MegaportLogger) Debug(args ...interface{}) {
	d.log(DebugLevel, args...)
}

// Emit the message and args at TRACE level
func (d *MegaportLogger) Trace(args ...interface{}) {
	d.log(TraceLevel, args...)
}

// Emit the message and args at INFO level
func (d *MegaportLogger) Info(args ...interface{}) {
	d.log(InfoLevel, args...)
}

// Emit the message and args at WARN level
func (d *MegaportLogger) Warn(args ...interface{}) {
	d.log(WarnLevel, args...)
}

// Emit the message and args at ERROR level
func (d *MegaportLogger) Error(args ...interface{}) {
	d.log(ErrorLevel, args...)
}

func (d *MegaportLogger) logf(level Level, format string, args ...interface{}) {
	if level >= d.level {
		msg := fmt.Sprintf(format, args...)
		log.Println(fmt.Sprintf("[%s] %s", strings.ToUpper(level.String()), msg))
	}
}

// Emit the message and args at DEBUG level
func (d *MegaportLogger) Debugf(format string, args ...interface{}) {
	d.logf(DebugLevel, format, args...)
}

// Emit the message and args at TRACE level
func (d *MegaportLogger) Tracef(format string, args ...interface{}) {
	d.logf(TraceLevel, format, args...)
}

// Emit the message and args at INFO level
func (d *MegaportLogger) Infof(format string, args ...interface{}) {
	d.logf(InfoLevel, format, args...)
}

// Emit the message and args at WARN level
func (d *MegaportLogger) Warnf(format string, args ...interface{}) {
	d.logf(WarnLevel, format, args...)
}

// Emit the message and args at ERROR level
func (d *MegaportLogger) Errorf(format string, args ...interface{}) {
	d.logf(ErrorLevel, format, args...)
}

func (d *MegaportLogger) logln(level Level, args ...interface{}) {
	if level >= d.level {
		msg := fmt.Sprintln(args...)
		prefix := fmt.Sprintf("[%s]", strings.ToUpper(level.String()))

		log.Println(prefix, msg[:len(msg)-1])
	}
}

// Emit the message and args at DEBUG level
func (d *MegaportLogger) Debugln(args ...interface{}) {
	d.logln(DebugLevel, args...)
}

// Emit the message and args at TRACE level
func (d *MegaportLogger) Traceln(args ...interface{}) {
	d.logln(TraceLevel, args...)
}

// Emit the message and args at INFO level
func (d *MegaportLogger) Infoln(args ...interface{}) {
	d.logln(InfoLevel, args...)
}

// Emit the message and args at WARN level
func (d *MegaportLogger) Warnln(args ...interface{}) {
	d.logln(WarnLevel, args...)
}

// Emit the message and args at ERROR level
func (d *MegaportLogger) Errorln(args ...interface{}) {
	d.logln(ErrorLevel, args...)
}
