package logme

import (
	"io"
	"log"
	"os"
	"path"
	"time"
)

var (
	info *log.Logger
	warn *log.Logger
	errs *log.Logger

	logFile *os.File
)

func Init(logsDir string) error {
	err := os.MkdirAll(logsDir, os.ModeDir|0755)
	if err != nil {
		return err
	}

	logFileNameFormat := time.Now().Format("2006-01-02 15_04_05")

	logFile, err = os.OpenFile(path.Join(logsDir, logFileNameFormat+".txt"), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	info = log.New(logFile, "[INFO]", log.LstdFlags)
	warn = log.New(io.MultiWriter(logFile, os.Stderr), "[WARN]", log.LstdFlags)
	errs = log.New(io.MultiWriter(logFile, os.Stderr), "[ERROR]", log.LstdFlags|log.Llongfile)
	return nil
}

func Info() *log.Logger {
	return info
}

func Warn() *log.Logger {
	return warn
}

func Err() *log.Logger {
	return errs
}

func Deinit() {
	if logFile != nil {
		logFile.Sync()
		logFile.Close()
	}
}
