package errors

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogLevel은 로그 레벨을 나타냅니다.
type LogLevel int

const (
	LogLevelSilent LogLevel = iota // 로그 출력 안함
	LogLevelError                  // 에러만 출력
	LogLevelWarn                   // 경고 이상 출력
	LogLevelInfo                   // 정보 이상 출력
	LogLevelDebug                  // 모든 로그 출력
)

// String은 LogLevel의 문자열 표현을 반환합니다.
func (l LogLevel) String() string {
	switch l {
	case LogLevelSilent:
		return "SILENT"
	case LogLevelError:
		return "ERROR"
	case LogLevelWarn:
		return "WARN"
	case LogLevelInfo:
		return "INFO"
	case LogLevelDebug:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel은 문자열을 LogLevel로 변환합니다.
func ParseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "SILENT":
		return LogLevelSilent
	case "ERROR":
		return LogLevelError
	case "WARN", "WARNING":
		return LogLevelWarn
	case "INFO":
		return LogLevelInfo
	case "DEBUG":
		return LogLevelDebug
	default:
		return LogLevelInfo
	}
}

// ErrorLogger는 에러 로깅을 담당하는 인터페이스입니다.
type ErrorLogger interface {
	LogError(err *CLIError)
	LogErrorWithLevel(level LogLevel, err *CLIError)
	SetLevel(level LogLevel)
	SetOutput(writer io.Writer)
	SetFormatter(formatter ErrorFormatter)
	Close() error
}

// FileErrorLogger는 파일 기반 에러 로거입니다.
type FileErrorLogger struct {
	level     LogLevel
	output    io.Writer
	file      *os.File
	formatter ErrorFormatter
	logger    *log.Logger
}

// NewFileErrorLogger는 새로운 파일 에러 로거를 생성합니다.
func NewFileErrorLogger(logPath string, level LogLevel) (*FileErrorLogger, error) {
	// 로그 디렉토리 생성
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("로그 디렉토리 생성 실패: %w", err)
	}
	
	// 로그 파일 열기 (append 모드)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("로그 파일 열기 실패: %w", err)
	}
	
	logger := log.New(file, "", 0) // 커스텀 포맷 사용을 위해 플래그 제거
	
	return &FileErrorLogger{
		level:     level,
		output:    file,
		file:      file,
		formatter: NewPlainErrorFormatter(), // 파일에는 플레인 포맷 사용
		logger:    logger,
	}, nil
}

// NewConsoleErrorLogger는 콘솔 에러 로거를 생성합니다.
func NewConsoleErrorLogger(level LogLevel, colorEnabled bool) *FileErrorLogger {
	formatter := NewHumanErrorFormatter(colorEnabled, true)
	logger := log.New(os.Stderr, "", 0)
	
	return &FileErrorLogger{
		level:     level,
		output:    os.Stderr,
		formatter: formatter,
		logger:    logger,
	}
}

// LogError는 에러를 로그에 기록합니다.
func (l *FileErrorLogger) LogError(err *CLIError) {
	l.LogErrorWithLevel(LogLevelError, err)
}

// LogErrorWithLevel은 지정된 레벨로 에러를 로그에 기록합니다.
func (l *FileErrorLogger) LogErrorWithLevel(level LogLevel, err *CLIError) {
	if l.level == LogLevelSilent || level > l.level {
		return
	}
	
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	// 로그 엔트리 생성
	var logEntry strings.Builder
	logEntry.WriteString(fmt.Sprintf("[%s] %s: ", timestamp, level.String()))
	
	// 에러 포맷팅
	if l.formatter != nil {
		// 상세 정보는 DEBUG 레벨에서만 출력
		verbose := l.level >= LogLevelDebug
		formatted := l.formatter.FormatWithDetails(err, verbose)
		
		// 멀티라인 로그를 위해 각 줄에 접두사 추가
		lines := strings.Split(strings.TrimSpace(formatted), "\n")
		for i, line := range lines {
			if i == 0 {
				logEntry.WriteString(line)
			} else {
				logEntry.WriteString(fmt.Sprintf("\n[%s] %s:   %s", timestamp, level.String(), line))
			}
		}
	} else {
		logEntry.WriteString(err.Error())
	}
	
	// 구분선 추가 (DEBUG 레벨에서만)
	if l.level >= LogLevelDebug {
		logEntry.WriteString(fmt.Sprintf("\n[%s] %s: ---", timestamp, level.String()))
	}
	
	// 로그 출력
	l.logger.Println(logEntry.String())
}

// SetLevel은 로그 레벨을 설정합니다.
func (l *FileErrorLogger) SetLevel(level LogLevel) {
	l.level = level
}

// SetOutput은 출력 대상을 설정합니다.
func (l *FileErrorLogger) SetOutput(writer io.Writer) {
	l.output = writer
	l.logger.SetOutput(writer)
}

// SetFormatter는 포맷터를 설정합니다.
func (l *FileErrorLogger) SetFormatter(formatter ErrorFormatter) {
	l.formatter = formatter
}

// Close는 로거를 종료하고 파일을 닫습니다.
func (l *FileErrorLogger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// MultiErrorLogger는 여러 로거에 동시에 로그를 기록합니다.
type MultiErrorLogger struct {
	loggers []ErrorLogger
}

// NewMultiErrorLogger는 새로운 멀티 에러 로거를 생성합니다.
func NewMultiErrorLogger(loggers ...ErrorLogger) *MultiErrorLogger {
	return &MultiErrorLogger{
		loggers: loggers,
	}
}

// AddLogger는 로거를 추가합니다.
func (m *MultiErrorLogger) AddLogger(logger ErrorLogger) {
	m.loggers = append(m.loggers, logger)
}

// LogError는 모든 로거에 에러를 기록합니다.
func (m *MultiErrorLogger) LogError(err *CLIError) {
	for _, logger := range m.loggers {
		logger.LogError(err)
	}
}

// LogErrorWithLevel은 모든 로거에 지정된 레벨로 에러를 기록합니다.
func (m *MultiErrorLogger) LogErrorWithLevel(level LogLevel, err *CLIError) {
	for _, logger := range m.loggers {
		logger.LogErrorWithLevel(level, err)
	}
}

// SetLevel은 모든 로거의 레벨을 설정합니다.
func (m *MultiErrorLogger) SetLevel(level LogLevel) {
	for _, logger := range m.loggers {
		logger.SetLevel(level)
	}
}

// SetOutput은 모든 로거의 출력을 설정합니다.
func (m *MultiErrorLogger) SetOutput(writer io.Writer) {
	for _, logger := range m.loggers {
		logger.SetOutput(writer)
	}
}

// SetFormatter는 모든 로거의 포맷터를 설정합니다.
func (m *MultiErrorLogger) SetFormatter(formatter ErrorFormatter) {
	for _, logger := range m.loggers {
		logger.SetFormatter(formatter)
	}
}

// Close는 모든 로거를 종료합니다.
func (m *MultiErrorLogger) Close() error {
	var errors []string
	for _, logger := range m.loggers {
		if err := logger.Close(); err != nil {
			errors = append(errors, err.Error())
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("로거 종료 중 오류 발생: %s", strings.Join(errors, ", "))
	}
	return nil
}

// GlobalErrorLogger는 전역 에러 로거입니다.
var GlobalErrorLogger ErrorLogger

// InitializeGlobalLogger는 전역 로거를 초기화합니다.
func InitializeGlobalLogger(logPath string, level LogLevel, enableConsole bool) error {
	var loggers []ErrorLogger
	
	// 파일 로거 추가 (항상)
	if logPath != "" {
		fileLogger, err := NewFileErrorLogger(logPath, level)
		if err != nil {
			return fmt.Errorf("파일 로거 초기화 실패: %w", err)
		}
		loggers = append(loggers, fileLogger)
	}
	
	// 콘솔 로거 추가 (선택적)
	if enableConsole {
		colorEnabled := os.Getenv("NO_COLOR") == ""
		consoleLogger := NewConsoleErrorLogger(level, colorEnabled)
		loggers = append(loggers, consoleLogger)
	}
	
	if len(loggers) == 1 {
		GlobalErrorLogger = loggers[0]
	} else if len(loggers) > 1 {
		GlobalErrorLogger = NewMultiErrorLogger(loggers...)
	} else {
		// 로거가 없으면 콘솔 로거 사용
		GlobalErrorLogger = NewConsoleErrorLogger(LogLevelError, false)
	}
	
	return nil
}

// LogError는 전역 로거를 사용하여 에러를 기록합니다.
func LogError(err *CLIError) {
	if GlobalErrorLogger != nil {
		GlobalErrorLogger.LogError(err)
	}
}

// LogErrorWithLevel은 전역 로거를 사용하여 지정된 레벨로 에러를 기록합니다.
func LogErrorWithLevel(level LogLevel, err *CLIError) {
	if GlobalErrorLogger != nil {
		GlobalErrorLogger.LogErrorWithLevel(level, err)
	}
}

// SetGlobalLogLevel은 전역 로거의 레벨을 설정합니다.
func SetGlobalLogLevel(level LogLevel) {
	if GlobalErrorLogger != nil {
		GlobalErrorLogger.SetLevel(level)
	}
}

// CloseGlobalLogger는 전역 로거를 종료합니다.
func CloseGlobalLogger() error {
	if GlobalErrorLogger != nil {
		return GlobalErrorLogger.Close()
	}
	return nil
}