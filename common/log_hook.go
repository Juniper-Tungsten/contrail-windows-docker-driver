package common

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

type LogToFileHook struct {
	Logfile   *os.File
	formatter *log.TextFormatter
}

func NewLogToFileHook(file *os.File) *LogToFileHook {
	return &LogToFileHook{
		Logfile:   file,
		formatter: new(log.TextFormatter),
	}
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

func LogFilepath() string {
	return string(filepath.Join(os.Getenv("ProgramData"), WinServiceName, "log.txt"))
}
