/*
 * LURE - Linux User REpository
 * Copyright (C) 2023 Elara Musayelyan
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package log

import (
	"os"

	"go.elara.ws/logger"
)

var Logger logger.Logger = logger.NewCLI(os.Stderr)

// NoPanic prevents the logger from panicking on panic events
func NoPanic() {
	Logger.NoPanic()
}

// NoExit prevents the logger from exiting on fatal events
func NoExit() {
	Logger.NoExit()
}

// SetLevel sets the log level of the logger
func SetLevel(l logger.LogLevel) {
	Logger.SetLevel(l)
}

// Debug creates a new debug event with the given message
func Debug(msg string) logger.LogBuilder {
	return Logger.Debug(msg)
}

// Debugf creates a new debug event with the formatted message
func Debugf(format string, v ...any) logger.LogBuilder {
	return Logger.Debugf(format, v...)
}

// Info creates a new info event with the given message
func Info(msg string) logger.LogBuilder {
	return Logger.Info(msg)
}

// Infof creates a new info event with the formatted message
func Infof(format string, v ...any) logger.LogBuilder {
	return Logger.Infof(format, v...)
}

// Warn creates a new warn event with the given message
func Warn(msg string) logger.LogBuilder {
	return Logger.Warn(msg)
}

// Warnf creates a new warn event with the formatted message
func Warnf(format string, v ...any) logger.LogBuilder {
	return Logger.Warnf(format, v...)
}

// Error creates a new error event with the given message
func Error(msg string) logger.LogBuilder {
	return Logger.Error(msg)
}

// Errorf creates a new error event with the formatted message
func Errorf(format string, v ...any) logger.LogBuilder {
	return Logger.Errorf(format, v...)
}

// Fatal creates a new fatal event with the given message
func Fatal(msg string) logger.LogBuilder {
	return Logger.Fatal(msg)
}

// Fatalf creates a new fatal event with the formatted message
func Fatalf(format string, v ...any) logger.LogBuilder {
	return Logger.Fatalf(format, v...)
}

// Fatal creates a new fatal event with the given message
func Panic(msg string) logger.LogBuilder {
	return Logger.Panic(msg)
}

// Fatalf creates a new fatal event with the formatted message
func Panicf(format string, v ...any) logger.LogBuilder {
	return Logger.Panicf(format, v...)
}
