package logging

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LogLevel int

const (
	LogLevelTrace LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarning
	LogLevelError
	LogLevelCritical
	LogLevelOff
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelTrace:
		return "TRACE"
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarning:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelCritical:
		return "CRITICAL"
	default:
		return "OFF"
	}
}

type LogService struct {
	logger  *log.Logger
	level   LogLevel
	logFile *os.File
	mu      sync.Mutex
	logDir  string
}

func NewLogService(logDir string) *LogService {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v, using stdout", err)
		return &LogService{
			logger: log.New(os.Stdout, "", log.LstdFlags),
			level:  LogLevelInfo,
		}
	}

	logPath := filepath.Join(logDir, "gode-"+time.Now().Format("2006-01-02")+".log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to open log file: %v, using stdout", err)
		return &LogService{
			logger: log.New(os.Stdout, "", log.LstdFlags),
			level:  LogLevelInfo,
		}
	}

	return &LogService{
		logger:  log.New(io.MultiWriter(os.Stdout, f), "", log.LstdFlags),
		level:   LogLevelInfo,
		logFile: f,
		logDir:  logDir,
	}
}

func (s *LogService) SetLevel(level LogLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.level = level
}

func (s *LogService) GetLevel() LogLevel {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.level
}

func (s *LogService) Trace(args ...interface{})      { s.log(LogLevelTrace, args...) }
func (s *LogService) Debug(args ...interface{})      { s.log(LogLevelDebug, args...) }
func (s *LogService) Info(args ...interface{})       { s.log(LogLevelInfo, args...) }
func (s *LogService) Warn(args ...interface{})       { s.log(LogLevelWarning, args...) }
func (s *LogService) Error(args ...interface{})      { s.log(LogLevelError, args...) }
func (s *LogService) Critical(args ...interface{})   { s.log(LogLevelCritical, args...) }

func (s *LogService) Logf(level LogLevel, format string, args ...interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if level < s.level {
		return
	}
	msg := level.String() + " " + time.Now().Format("2006-01-02 15:04:05.000") + " " + format
	s.logger.Printf(msg, args...)
}

func (s *LogService) log(level LogLevel, args ...interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if level < s.level {
		return
	}
	msg := level.String() + " " + time.Now().Format("2006-01-02 15:04:05.000")
	s.logger.Println(append([]interface{}{msg}, args...)...)
}

func (s *LogService) Dispose() {
	if s.logFile != nil {
		s.logFile.Close()
	}
}