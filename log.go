package log4go

import (
	"bytes"
	"fmt"
	"log"
	"path"
	"runtime"
	"sync"
	"time"
)

// LevelFlags log level flags
const (
	LevelFlagsEmergency     = "EMERGENCY"
	LevelFlagsAlert         = "ALERT"
	LevelFlagsCritical      = "CRITICAL"
	LevelFlagsError         = "ERROR"
	LevelFlagsWarning       = "WARNING"
	LevelFlagsNotice        = "NOTICE"
	LevelFlagsInformational = "INFO"
	LevelFlagsDebug         = "DEBUG"
)

// RFC5424 log message levels.
// ref: https://tools.ietf.org/html/draft-ietf-syslog-protocol-23
const (
	EMERGENCY = iota // Emergency: system is unusable
	ALERT            // Alert: action must be taken immediately
	CRITICAL         // Critical: critical conditions
	ERROR            // Error: error conditions
	WARNING          // Warning: warning conditions
	NOTICE           // Notice: normal but significant condition
	INFO             // Informational: informational messages
	DEBUG            // Debug: debug-level messages
)

const (
	// default size or min size for record channel
	recordChannelSizeDefault = uint(2048)

	// default time layout
	defaultLayout = "2006/01/02 15:04:05"
)

// LevelFlags level Flags set
var (
	LevelFlags = []string{
		LevelFlagsEmergency,
		LevelFlagsAlert,
		LevelFlagsCritical,
		LevelFlagsError,
		LevelFlagsWarning,
		LevelFlagsNotice,
		LevelFlagsInformational,
		LevelFlagsDebug,
	}
	DefaultLayout = defaultLayout
)

// default looger
var (
	loggerDefault     *Logger
	recordPool        *sync.Pool
	recordChannelSize = recordChannelSizeDefault // log chan size
)

// Record log record
type Record struct {
	time  string
	level int
	file  string
	msg   string
}

func (r *Record) String() string {
	return fmt.Sprintf("%s [%s] <%s> %s\n", r.time, LevelFlags[r.level], r.file, r.msg)
}

// Writer record writer
type Writer interface {
	Init() error
	Write(*Record) error
}

//Flusher record flusher
type Flusher interface {
	Flush() error
}

// Rotater record rotater
type Rotater interface {
	Rotate() error
	SetPathPattern(string) error
}

// Logger logger define
type Logger struct {
	writers         []Writer
	records         chan *Record
	recordsChanSize uint
	lastTime        int64
	lastTimeStr     string

	flushTimer  time.Duration // timer to flush logger record to chan
	rotateTimer time.Duration // timer to rotate logger record for writer

	c chan bool

	layout       string
	level        int
	fullPath     bool // show full path, default only show file:line_number
	withFuncName bool // show caller func name
	lock         sync.RWMutex
}

// NewLogger create the logger
func NewLogger() *Logger {
	if loggerDefault != nil {
		return loggerDefault
	}
	records := make(chan *Record, recordChannelSize)

	return newLoggerWithRecords(records)
}

// newLoggerWithRecords is useful for go test
func newLoggerWithRecords(records chan *Record) *Logger {
	l := new(Logger)
	l.writers = make([]Writer, 0, 1) // normal least has console writer
	if l.recordsChanSize == 0 {
		recordChannelSize = recordChannelSizeDefault
	}

	l.records = records
	l.c = make(chan bool, 1)
	l.level = DEBUG
	l.layout = DefaultLayout

	go boostrapLogWriter(l)

	return l
}

// Register register writer
// the writer should be register once for writers by kind
func (l *Logger) Register(w Writer) {
	if err := w.Init(); err != nil {
		panic(err)
	}

	l.writers = append(l.writers, w)
}

// Close close logger
func (l *Logger) Close() {
	close(l.records)
	<-l.c

	for _, w := range l.writers {
		if f, ok := w.(Flusher); ok {
			if err := f.Flush(); err != nil {
				log.Println(err)
			}
		}
	}
}

// SetLayout set the logger time layout
func (l *Logger) SetLayout(layout string) {
	l.layout = layout
}

// SetLevel set the logger level
func (l *Logger) SetLevel(lvl int) {
	l.level = lvl
}

// WithFullPath set the logger with full path
func (l *Logger) WithFullPath(show bool) {
	l.fullPath = show
}

// WithFuncName set the logger with func name
func (l *Logger) WithFuncName(show bool) {
	l.withFuncName = show
}

// Debug level debug
func (l *Logger) Debug(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(DEBUG, fmt, args...)
}

// Info level info
func (l *Logger) Info(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(INFO, fmt, args...)
}

// Notice level notice
func (l *Logger) Notice(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(NOTICE, fmt, args...)
}

// Warn level warn
func (l *Logger) Warn(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(WARNING, fmt, args...)
}

// Error level error
func (l *Logger) Error(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(ERROR, fmt, args...)
}

// Critical level critical
func (l *Logger) Critical(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(CRITICAL, fmt, args...)
}

// Alert level alert
func (l *Logger) Alert(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(ALERT, fmt, args...)
}

// Emergency level emergency
func (l *Logger) Emergency(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(EMERGENCY, fmt, args...)
}

func (l *Logger) deliverRecordToWriter(level int, format string, args ...interface{}) {
	var msg string
	var fi bytes.Buffer

	if level > l.level {
		return
	}

	if format != "" {
		msg = fmt.Sprintf(format, args...)
	} else {
		msg = fmt.Sprint(args...)
	}

	// source code, file and line num
	pc, file, line, ok := runtime.Caller(2)
	if ok {
		fileName := path.Base(file)
		if l.fullPath {
			fileName = file
		}
		fi.WriteString(fmt.Sprintf("%s:%d", fileName, line))

		if l.withFuncName {
			funcName := runtime.FuncForPC(pc).Name()
			funcName = path.Base(funcName)
			fi.WriteString(fmt.Sprintf(" %s", funcName))
		}
	}

	// format time
	now := time.Now()
	l.lock.Lock() // avoid data race
	if now.Unix() != l.lastTime {
		l.lastTime = now.Unix()
		l.lastTimeStr = now.Format(l.layout)
	}
	lastTimeStr := l.lastTimeStr
	l.lock.Unlock()

	r := recordPool.Get().(*Record)
	r.msg = msg
	r.file = fi.String()
	r.time = lastTimeStr
	r.level = level

	l.records <- r
}

func boostrapLogWriter(logger *Logger) {
	var (
		r  *Record
		ok bool
	)

	if r, ok = <-logger.records; !ok {
		logger.c <- true
		return
	}

	for _, w := range logger.writers {
		if err := w.Write(r); err != nil {
			log.Printf("%v\n", err)
		}
	}

	flushTimer := time.NewTimer(logger.flushTimer)
	rotateTimer := time.NewTimer(logger.rotateTimer)

	for {
		select {
		case r, ok = <-logger.records:
			if !ok {
				logger.c <- true
				return
			}

			for _, w := range logger.writers {
				if err := w.Write(r); err != nil {
					log.Printf("%v\n", err)
				}
			}

			recordPool.Put(r)

		case <-flushTimer.C:
			for _, w := range logger.writers {
				if f, ok := w.(Flusher); ok {
					if err := f.Flush(); err != nil {
						log.Printf("%v\n", err)
					}
				}
			}
			flushTimer.Reset(logger.flushTimer)

		case <-rotateTimer.C:
			for _, w := range logger.writers {
				if r, ok := w.(Rotater); ok {
					if err := r.Rotate(); err != nil {
						log.Printf("%v\n", err)
					}
				}
			}
			rotateTimer.Reset(logger.rotateTimer)
		}
	}
}

func init() {
	loggerDefault = NewLogger()
	loggerDefault.flushTimer = time.Millisecond * 500
	loggerDefault.rotateTimer = time.Second * 10
	recordPool = &sync.Pool{New: func() interface{} {
		return &Record{}
	}}
}

// Register register writer
func Register(w Writer) {
	loggerDefault.Register(w)
}

// Close close logger
func Close() {
	loggerDefault.Close()
}

// SetLayout set the logger time layout, should call before logger real use
func SetLayout(layout string) {
	loggerDefault.layout = layout
}

// SetLevel set the logger level, should call before logger real use
func SetLevel(lvl int) {
	loggerDefault.level = lvl
}

// WithFullPath set the logger with full path, should call before logger real use
func WithFullPath(show bool) {
	loggerDefault.fullPath = show
}

// WithFuncName set the logger with func name, should call before logger real use
func WithFuncName(show bool) {
	loggerDefault.withFuncName = show
}

// Debug level debug
func Debug(fmt string, args ...interface{}) {
	loggerDefault.deliverRecordToWriter(DEBUG, fmt, args...)
}

// Info level info
func Info(fmt string, args ...interface{}) {
	loggerDefault.deliverRecordToWriter(INFO, fmt, args...)
}

// Notice level notice
func Notice(fmt string, args ...interface{}) {
	loggerDefault.deliverRecordToWriter(NOTICE, fmt, args...)
}

// Warn level warn
func Warn(fmt string, args ...interface{}) {
	loggerDefault.deliverRecordToWriter(WARNING, fmt, args...)
}

// Error level error
func Error(fmt string, args ...interface{}) {
	loggerDefault.deliverRecordToWriter(ERROR, fmt, args...)
}

// Critical level critical
func Critical(fmt string, args ...interface{}) {
	loggerDefault.deliverRecordToWriter(CRITICAL, fmt, args...)
}

// Alert level alert
func Alert(fmt string, args ...interface{}) {
	loggerDefault.deliverRecordToWriter(ALERT, fmt, args...)
}

// Emergency level emergency
func Emergency(fmt string, args ...interface{}) {
	loggerDefault.deliverRecordToWriter(EMERGENCY, fmt, args...)
}
