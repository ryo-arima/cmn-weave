package share

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ryo-arima/cmn-weave/pkg/global"
)

// ServerLoggerConfig holds configuration for the server logger.
type ServerLoggerConfig struct {
	Component    string `json:"component"    yaml:"component"`
	Service      string `json:"service"      yaml:"service"`
	Level        string `json:"level"        yaml:"level"`
	Structured   bool   `json:"structured"   yaml:"structured"`
	EnableCaller bool   `json:"enable_caller" yaml:"enable_caller"`
	Output       string `json:"output"       yaml:"output"`
}

// ServerLoggerInterface is the logging contract used throughout the server.
type ServerLoggerInterface interface {
	DEBUG(requestID string, mcode global.MCode, optionalMessage string, fields ...map[string]interface{})
	INFO(requestID string, mcode global.MCode, optionalMessage string)
	WARN(requestID string, mcode global.MCode, optionalMessage string)
	ERROR(requestID string, mcode global.MCode, optionalMessage string)
	FATAL(requestID string, mcode global.MCode, optionalMessage string)
}

// ServerLogLevel represents the log level.
type ServerLogLevel int

const (
	SERVER_DEBUG ServerLogLevel = iota
	SERVER_INFO
	SERVER_WARN
	SERVER_ERROR
	SERVER_FATAL
)

func (rcvr ServerLogLevel) String() string {
	switch rcvr {
	case SERVER_DEBUG:
		return "DEBUG  "
	case SERVER_INFO:
		return "INFO   "
	case SERVER_WARN:
		return "WARN   "
	case SERVER_ERROR:
		return "ERROR  "
	case SERVER_FATAL:
		return "FATAL  "
	default:
		return "UNKNOWN"
	}
}

// ServerLogEntry is a single structured log record.
type ServerLogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Code      string                 `json:"code"`
	Component string                 `json:"component"`
	Service   string                 `json:"service"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	File      string                 `json:"file,omitempty"`
	Function  string                 `json:"function,omitempty"`
	Line      int                    `json:"line,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// ServerLogger is the concrete logger for the server component.
type ServerLogger struct {
	config *ServerLoggerConfig
	level  ServerLogLevel
	output io.Writer
}

// globalServerLogger is the process-wide server logger instance.
var globalServerLogger *ServerLogger

// GetServerLogger returns the global server logger.
// Returns a default stdout logger if SetServerLogger has not been called yet.
func GetServerLogger() *ServerLogger {
	if globalServerLogger == nil {
		globalServerLogger = defaultServerLogger()
	}
	return globalServerLogger
}

// SetServerLogger sets the global server logger instance.
func SetServerLogger(logger *ServerLogger) {
	globalServerLogger = logger
}

func defaultServerLogger() *ServerLogger {
	return &ServerLogger{
		config: &ServerLoggerConfig{Component: "server", Level: "info"},
		level:  SERVER_INFO,
		output: os.Stdout,
	}
}

// NewServerLogger creates and returns a new ServerLogger.
func NewServerLogger(cfg ServerLoggerConfig) *ServerLogger {
	l := &ServerLogger{
		config: &cfg,
		output: os.Stdout,
	}
	switch strings.ToUpper(cfg.Level) {
	case "DEBUG":
		l.level = SERVER_DEBUG
	case "INFO":
		l.level = SERVER_INFO
	case "WARN":
		l.level = SERVER_WARN
	case "ERROR":
		l.level = SERVER_ERROR
	case "FATAL":
		l.level = SERVER_FATAL
	default:
		l.level = SERVER_INFO
	}
	switch cfg.Output {
	case "stderr":
		l.output = os.Stderr
	case "stdout", "":
		l.output = os.Stdout
	default:
		if f, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666); err == nil {
			l.output = f
		} else {
			l.output = os.Stdout
			fmt.Fprintf(os.Stderr, "failed to open log file %s: %v\n", cfg.Output, err)
		}
	}
	return l
}

func (rcvr *ServerLogger) log(level ServerLogLevel, requestID string, mcode global.MCode, optionalMessage string, fields map[string]interface{}) {
	if level < rcvr.level {
		return
	}
	msg := mcode.Message
	if optionalMessage != "" {
		msg = fmt.Sprintf("%s: %s", mcode.Message, optionalMessage)
	}
	entry := ServerLogEntry{
		Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05.000000000") + " UTC",
		Level:     level.String(),
		Code:      mcode.PaddedCode(),
		Component: rcvr.config.Component,
		Service:   rcvr.config.Service,
		Message:   msg,
		Fields:    fields,
		RequestID: requestID,
	}
	rcvr.writeLogEntry(entry)
}

func (rcvr *ServerLogger) writeLogEntry(entry ServerLogEntry) {
	if rcvr.config.EnableCaller || rcvr.level == SERVER_DEBUG {
		if pc, file, line, ok := runtime.Caller(4); ok {
			entry.File = file
			entry.Line = line
			if fn := runtime.FuncForPC(pc); fn != nil {
				entry.Function = fn.Name()
			}
		}
	}
	if entry.Fields != nil {
		if err, ok := entry.Fields["error"].(string); ok {
			entry.Error = err
			delete(entry.Fields, "error")
		}
		if err, ok := entry.Fields["error"].(error); ok {
			entry.Error = err.Error()
			delete(entry.Fields, "error")
		}
	}
	if rcvr.config.Structured {
		if b, err := json.Marshal(entry); err == nil {
			fmt.Fprintln(rcvr.output, string(b))
		} else {
			fmt.Fprintf(rcvr.output, "[%s] [%s] [%s] %s\n", entry.Timestamp, entry.Level, entry.Code, entry.Message)
		}
		return
	}
	rid := entry.RequestID
	if rid == "" {
		rid = "xxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	}
	fmt.Fprintf(rcvr.output, "[%s] [%s] [%s] [%s] %s", entry.Timestamp, entry.Level, entry.Code, rid, entry.Message)
	if len(entry.Fields) > 0 && entry.Level == "DEBUG  " {
		if b, err := json.Marshal(entry.Fields); err == nil {
			fmt.Fprintf(rcvr.output, " %s", string(b))
		}
	}
	fmt.Fprintln(rcvr.output)
}

func (rcvr *ServerLogger) DEBUG(requestID string, mcode global.MCode, optionalMessage string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	rcvr.log(SERVER_DEBUG, requestID, mcode, optionalMessage, f)
}

func (rcvr *ServerLogger) INFO(requestID string, mcode global.MCode, optionalMessage string) {
	rcvr.log(SERVER_INFO, requestID, mcode, optionalMessage, nil)
}

func (rcvr *ServerLogger) WARN(requestID string, mcode global.MCode, optionalMessage string) {
	rcvr.log(SERVER_WARN, requestID, mcode, optionalMessage, nil)
}

func (rcvr *ServerLogger) ERROR(requestID string, mcode global.MCode, optionalMessage string) {
	rcvr.log(SERVER_ERROR, requestID, mcode, optionalMessage, nil)
}

func (rcvr *ServerLogger) FATAL(requestID string, mcode global.MCode, optionalMessage string) {
	rcvr.log(SERVER_FATAL, requestID, mcode, optionalMessage, nil)
	os.Exit(1)
}

// GinLoggerWriter wraps ServerLogger to implement io.Writer for Gin's output.
type GinLoggerWriter struct {
	logger ServerLoggerInterface
}

// NewGinLoggerWriter creates a GinLoggerWriter.
func NewGinLoggerWriter(logger ServerLoggerInterface) *GinLoggerWriter {
	return &GinLoggerWriter{logger: logger}
}

// Write implements io.Writer for Gin.
func (rcvr *GinLoggerWriter) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\n")
	if msg == "" {
		return len(p), nil
	}
	// Skip Gin's default HTTP access lines (they are handled by LoggerWithConfig).
	if strings.HasPrefix(msg, "[GIN] ") && strings.Contains(msg, " | ") {
		return len(p), nil
	}
	mcode := global.MCode{Code: "GINLOG", Message: ""}
	cleanMsg := msg
	for _, prefix := range []string{"[GIN-debug] ", "[GIN-warning] ", "[GIN-error] ", "[GIN] "} {
		cleanMsg = strings.TrimPrefix(cleanMsg, prefix)
	}
	cleanMsg = strings.TrimPrefix(cleanMsg, ": ")
	if strings.Contains(msg, "[WARNING]") || strings.Contains(msg, "[GIN-warning]") {
		rcvr.logger.WARN("", mcode, cleanMsg)
	} else if strings.Contains(msg, "[ERROR]") || strings.Contains(msg, "[GIN-error]") {
		rcvr.logger.ERROR("", mcode, cleanMsg)
	} else {
		rcvr.logger.INFO("", mcode, cleanMsg)
	}
	return len(p), nil
}

// LoggerWithConfig returns a Gin middleware that logs each HTTP request.
func LoggerWithConfig(logger ServerLoggerInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		c.Next()
		if raw != "" {
			path = path + "?" + raw
		}
		requestID := GetRequestID(c)
		latency := time.Since(start)
		fields := map[string]interface{}{
			"method":     c.Request.Method,
			"path":       path,
			"client_ip":  c.ClientIP(),
			"status":     c.Writer.Status(),
			"latency_ms": latency.Milliseconds(),
		}
		if len(c.Errors) > 0 {
			fields["error"] = c.Errors.Last().Error()
		}
		status := c.Writer.Status()
		info := fmt.Sprintf("%s %s %d", c.Request.Method, path, status)
		switch {
		case status >= 500:
			logger.ERROR(requestID, global.SMLWC5, info)
		case status >= 400:
			logger.WARN(requestID, global.SMLWC4, info)
		default:
			logger.INFO(requestID, global.SMLWC3, info)
		}
	}
}
