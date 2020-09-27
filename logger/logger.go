package logger

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/kahgeh/devenv/lang"
	"github.com/theckman/yacspin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogLevel uint8

const (
	NormalLogLevel   = 1
	DetailedLogLevel = 2
	DebugLogLevel    = 3
)

const ExitFailureStatus = 3

type loggerState struct {
	defaultLogger      *Spinner
	detailedLoggerBase *zap.Logger
	detailedLogger     *zap.SugaredLogger
	names              []string
	level              LogLevel
}

var state *loggerState

func newConsoleEncoderConfig(callerKey string) zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		// Keys can be anything except the empty string.
		TimeKey:        zapcore.OmitKey,
		LevelKey:       zapcore.OmitKey,
		NameKey:        "N",
		CallerKey:      callerKey,
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// newConsoleConfig is a reasonable console logging configuration.
// Logging is enabled at DebugLevel and above.
//
// It enables console mode (which makes DPanicLevel logs panic), uses a
// console encoder, writes to standard error, and disables sampling.
// Stacktraces are automatically included on logs of WarnLevel and above.
func newConsoleConfig(level zapcore.Level, callerKey string) zap.Config {
	return zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      true,
		Encoding:         "console",
		EncoderConfig:    newConsoleEncoderConfig(callerKey),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
}

func NewSpinner() *yacspin.Spinner {
	cfg := yacspin.Config{
		Frequency:         100 * time.Millisecond,
		CharSet:           yacspin.CharSets[37],
		Suffix:            "",
		SuffixAutoColon:   true,
		StopCharacter:     "✓",
		StopFailCharacter: "✗",
		StopColors:        []string{"fgGreen"},
		StopFailColors:    []string{"fgRed"},
	}

	spinner, _ := yacspin.New(cfg)
	return spinner
}

func GetCallerFunctionName() string {
	// Skip GetCallerFunctionName and the function to get the caller of
	fullFuncName := getFrame(2).Function
	parts := strings.Split(fullFuncName, "/")
	parts = strings.Split(parts[len(parts)-1], ".")
	return parts[len(parts)-1]
}

func getFrame(skipFrames int) runtime.Frame {
	targetFrameIndex := skipFrames + 2
	programCounters := make([]uintptr, targetFrameIndex+2)
	n := runtime.Callers(0, programCounters)

	frame := runtime.Frame{Function: "unknown"}
	if n > 0 {
		frames := runtime.CallersFrames(programCounters[:n])
		for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
			var frameCandidate runtime.Frame
			frameCandidate, more = frames.Next()
			if frameIndex == targetFrameIndex {
				frame = frameCandidate
			}
		}
	}

	return frame
}

func CreateLogger(level LogLevel) {
	if level == DetailedLogLevel {
		detailedLoggerBase, _ := newConsoleConfig(zapcore.InfoLevel, zapcore.OmitKey).Build()
		state = &loggerState{
			detailedLoggerBase: detailedLoggerBase,
			detailedLogger:     detailedLoggerBase.WithOptions(zap.AddCallerSkip(1)).Sugar(),
			defaultLogger:      nil,
			level:              level,
		}
		return
	}

	if level == DebugLogLevel {
		detailedLoggerBase, _ := newConsoleConfig(zapcore.DebugLevel, "caller").Build()
		state = &loggerState{
			detailedLoggerBase: detailedLoggerBase,
			detailedLogger:     detailedLoggerBase.WithOptions(zap.AddCallerSkip(1)).Sugar(),
			defaultLogger:      nil,
			level:              level,
		}
		return
	}

	if level == NormalLogLevel {
		spinner := &Spinner{
			logStream:    make(chan string),
			newComponent: make(chan *yacspin.Spinner),
		}
		state = &loggerState{
			level:              level,
			defaultLogger:      spinner,
			detailedLoggerBase: nil,
			detailedLogger:     nil,
		}
		go spinner.run()
		return
	}
}

func Sync() {
	if state.level > NormalLogLevel {
		state.detailedLoggerBase.Sync()
		state.detailedLogger.Sync()
	}
}

type Logger struct {
	defaultLogger  *Spinner
	detailedLogger *zap.SugaredLogger
	level          LogLevel
	LogDone        func()
}

func toLower(words []string) []string {
	var lowerCasedWords []string
	for _, word := range words {
		lowerCasedWords = append(lowerCasedWords, strings.ToLower(word))
	}
	return lowerCasedWords
}

func getName() string {
	fullName := strings.Join(state.names, ".")
	if len(fullName) > 20 && len(state.names) > 1 {
		lastName := state.names[len(state.names)-1]
		placeHolders := strings.Repeat(".", len(state.names)-1)
		fullName = fmt.Sprintf("%s%s", placeHolders, lastName)
	}
	padding := ""
	columnWidth := 20
	if len(fullName) < columnWidth {
		padding = strings.Repeat(" ", columnWidth-len(fullName))
	}
	return fmt.Sprintf("[%s]%s", fullName, padding)
}

func NewTaskLogger() *Logger {
	funcName := GetCallerFunctionName()
	words := toLower(lang.ToSentence(funcName))
	opName := lang.ToPresentParticiple(words)
	state.names = append(state.names, funcName)
	if state.level == NormalLogLevel {
		spinnerComponent := NewSpinner()
		failMessage := fmt.Sprintf("Failed to %s", strings.Join(words, " "))
		successMessage := fmt.Sprintf("Successfully %s", lang.ToPastTensePhrase(words))
		spinnerComponent.Message(opName)
		spinnerComponent.StopMessage(successMessage)
		spinnerComponent.StopFailMessage(failMessage)
		state.defaultLogger.newComponent <- spinnerComponent
		return &Logger{
			defaultLogger: state.defaultLogger,
			LogDone: func() {
				removeFuncName()
			},
		}
	}

	detailedLogger := state.detailedLogger.Named(getName())
	if state.level == DebugLogLevel {
		detailedLogger.Debug(opName)
	}
	return &Logger{
		level:          state.level,
		detailedLogger: detailedLogger,
		LogDone: func() {
			if state.level == DebugLogLevel {
				detailedLogger.Debugf("done %s", opName)
			}
			removeFuncName()
		},
	}
}

func New() *Logger {
	funcName := GetCallerFunctionName()
	words := lang.ToSentence(funcName)
	opName := lang.ToPresentParticiple(words)
	state.names = append(state.names, funcName)
	if state.level == NormalLogLevel {
		return &Logger{
			defaultLogger: state.defaultLogger,
			LogDone: func() {
				removeFuncName()
			},
		}
	}
	detailedLogger := state.detailedLogger.Named(getName())
	if state.level == DebugLogLevel {
		detailedLogger.Debugf("%s", opName)
	}
	return &Logger{
		level:          state.level,
		detailedLogger: detailedLogger,
		defaultLogger:  state.defaultLogger,
		LogDone: func() {
			if state.level == DebugLogLevel {
				detailedLogger.Debugf("done %s", opName)
			}
			removeFuncName()
		},
	}
}

func removeFuncName() {
	if len(state.names) > 0 {
		state.names = state.names[:len(state.names)-1]
	}
}

func (logger *Logger) Infof(template string, args ...interface{}) {
	if logger.defaultLogger != nil {
		logger.defaultLogger.update(fmt.Sprintf(template, args...))
		return
	}
	logger.detailedLogger.Infof(template, args...)
}

func (logger *Logger) Info(args ...interface{}) {
	if logger.defaultLogger != nil {
		logger.defaultLogger.update(fmt.Sprintf("%v", args...))
		return
	}

	logger.detailedLogger.Info(args...)
}

func (logger *Logger) Debugf(template string, args ...interface{}) {
	if logger.defaultLogger != nil {
		return
	}
	logger.detailedLogger.Debugf(template, args...)
}

func (logger *Logger) Debug(args ...interface{}) {
	if logger.defaultLogger != nil {
		return
	}

	logger.detailedLogger.Debug(args...)
}

func (logger *Logger) DebugFunc(log func(), mustExecute func()) {

	if logger.defaultLogger != nil {
		mustExecute()
		return
	}

	log()
}

func (logger *Logger) Fail(args ...interface{}) {
	if state.defaultLogger != nil {
		state.defaultLogger.failed(fmt.Sprintf("%v", args...))
		os.Exit(ExitFailureStatus)
		return
	}
	logger.detailedLogger.Error(args...)
	os.Exit(ExitFailureStatus)
}

func (logger *Logger) Failf(template string, args ...interface{}) {
	if state.defaultLogger != nil {
		state.defaultLogger.failed(fmt.Sprintf(template, args...))
		os.Exit(ExitFailureStatus)
		return
	}
	logger.detailedLogger.Errorf(template, args)
	os.Exit(ExitFailureStatus)
}

func (logger *Logger) Succeed() {
	if logger.defaultLogger != nil {
		state.defaultLogger.succeed()
	}
}

func (logger *Logger) Succeedf(template string, args ...interface{}) {
	if logger.defaultLogger != nil {
		message := fmt.Sprintf(template, args...)
		state.defaultLogger.update(message)
		state.defaultLogger.succeed()
		return
	}

	logger.detailedLogger.Infof(template, args...)
}
