//
// Copyright (c) 2018 Juniper Networks, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logging

import (
	"os"

	log "github.com/sirupsen/logrus"
)

type LogToFileHook struct {
	Logfile   *os.File
	formatter *log.TextFormatter
}

func NewLogToFileHook(file *os.File) *LogToFileHook {
	return &LogToFileHook{
		Logfile:   file,
		formatter: &log.TextFormatter{FullTimestamp: true},
	}
}

func (h *LogToFileHook) Close() {
	h.Logfile.Close()
}

func (h *LogToFileHook) Levels() []log.Level {
	return log.AllLevels
}

func (h *LogToFileHook) Fire(entry *log.Entry) (err error) {
	line, err := h.formatter.Format(entry)
	if err == nil {
		_, err = h.Logfile.WriteString(string(line))
		return err
	}
	return nil
}
