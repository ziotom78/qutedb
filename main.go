/*
The MIT License

Copyright (c) 2018 Maurizio Tomasi

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package main

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

// configureLogging sets up the Logrus library in order to use the
// desired format and verbosity level
func configureLogging(config *Configuration) {
	formatter := strings.ToLower(config.LogFormat)
	switch formatter {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	case "text", "default":
		log.SetFormatter(&log.TextFormatter{})
	default:
		panic(fmt.Errorf("unknown formatter: \"%s\"", formatter))
	}

	level := strings.ToLower(config.LogLevel)
	switch level {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn", "warning":
		log.SetLevel(log.WarnLevel)
	case "info", "default":
		log.SetLevel(log.InfoLevel)
	case "debug", "verbose":
		log.SetLevel(log.DebugLevel)
	}
}

func main() {
	InitApp()
	RunApp()
}
