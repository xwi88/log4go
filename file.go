package log4go

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var pathVariableTable map[byte]func(*time.Time) int

// FileWriter file writer for log record deal
type FileWriter struct {
	// write log order by order and atomic incr
	// maxLinesCurLines and maxSizeCurSize
	lock         sync.RWMutex
	initFileOnce sync.Once // init once
	initFileOk   bool

	level int

	// input filename
	filename string
	// The opened file
	file          *os.File
	fileBufWriter *bufio.Writer
	// like "xwi88.log", xwi88 is filenameOnly and .log is suffix
	filenameOnly, suffix string

	pathFmt   string // Rotate when, use actions
	actions   []func(*time.Time) int
	variables []interface{}

	rotate     bool
	perm       string      // input
	rotatePerm os.FileMode // real used

	// // Rotate at file lines
	// maxLines         int // Rotate at line
	// maxLinesCurLines int

	// // Rotate at size
	// maxSize        int
	// maxSizeCurSize int

	lastWriteTime time.Time
	// Rotate daily
	daily         bool
	maxDays       int
	dailyOpenDate int
	dailyOpenTime time.Time

	// Rotate hourly
	hourly         bool
	maxHours       int
	hourlyOpenDate int
	hourlyOpenTime time.Time

	// Rotate minutely
	minutely         bool
	maxMinutes       int
	minutelyOpenDate int
	minutelyOpenTime time.Time
}

// FileWriterOptions file writer options
type FileWriterOptions struct {
	Level    int
	Filename string

	Rotate bool
	// Rotate daily
	Daily   bool
	MaxDays int

	// Rotate hourly
	Hourly   bool
	MaxHours int

	// Rotate minutely
	Minutely   bool
	MaxMinutes int
}

// NewFileWriter create new file writer
func NewFileWriter() *FileWriter {
	return &FileWriter{}
}

// NewFileWriterWithOptions create new file writer with options
func NewFileWriterWithOptions(option FileWriterOptions) *FileWriter {
	defaultLevel := DEBUG
	if option.Level <= defaultLevel {
		defaultLevel = option.Level
	}

	return &FileWriter{
		level:      defaultLevel,
		filename:   option.Filename,
		rotate:     option.Rotate,
		daily:      option.Daily,
		maxDays:    option.MaxDays,
		hourly:     option.Hourly,
		maxHours:   option.MaxHours,
		minutely:   option.Minutely,
		maxMinutes: option.MaxMinutes,
	}
}

// Write file write
func (w *FileWriter) Write(r *Record) error {
	if r.level > w.level {
		return nil
	}
	if w.fileBufWriter == nil {
		return errors.New("FileWriter no opened file")
	}
	_, err := w.fileBufWriter.WriteString(r.String())
	return err
}

// Init file writer init
func (w *FileWriter) Init() error {
	filename := w.filename
	defaultPerm := "0644"
	if len(filename) != 0 {
		w.suffix = filepath.Ext(filename)
		w.filenameOnly = strings.TrimSuffix(filename, w.suffix)
		w.filename = filename
		if w.suffix == "" {
			w.suffix = ".log"
		}
	}
	if w.perm == "" {
		w.perm = defaultPerm
	}

	perm, err := strconv.ParseInt(w.perm, 8, 64)
	if err != nil {
		return err
	}
	w.rotatePerm = os.FileMode(perm)

	if w.rotate {
		if w.daily && w.maxDays <= 0 {
			w.maxDays = 1
		}
		if w.hourly && w.maxHours <= 0 {
			w.maxHours = 1
		}
		if w.minutely && w.maxMinutes <= 0 {
			w.maxMinutes = 1
		}
	}

	return w.Rotate()
}

// Flush writes any buffered data to file
func (w *FileWriter) Flush() error {
	if w.fileBufWriter != nil {
		return w.fileBufWriter.Flush()
	}
	return nil
}

// SetPathPattern for file writer
func (w *FileWriter) SetPathPattern(pattern string) error {
	n := 0
	for _, c := range pattern {
		if c == '%' {
			n++
		}
	}

	if n == 0 {
		w.pathFmt = pattern
		return nil
	}

	w.actions = make([]func(*time.Time) int, 0, n)
	w.variables = make([]interface{}, n, n)
	tmp := []byte(pattern)

	variable := 0
	for _, c := range tmp {
		if variable == 1 {
			act, ok := pathVariableTable[c]
			if !ok {
				return errors.New("Invalid rotate pattern (" + pattern + ")")
			}
			w.actions = append(w.actions, act)
			variable = 0
			continue
		}
		if c == '%' {
			variable = 1
		}
	}

	w.pathFmt = convertPatternToFmt(tmp)

	return nil
}

func (w *FileWriter) initFile() {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.initFileOk = true
}

// Rotate file writer rotate
func (w *FileWriter) Rotate() error {
	now := time.Now()
	v := 0
	rotate := false

	for i, act := range w.actions {
		v = act(&now)
		if v != w.variables[i] {
			if !w.initFileOk {
				w.variables[i] = v
				rotate = true
			} else {
				// only exec except the first round
				switch i {
				case 2:
					if w.daily {
						w.dailyOpenDate = v
						w.dailyOpenTime = now
						_, _, d := w.lastWriteTime.AddDate(0, 0, w.maxDays).Date()
						if v == d {
							rotate = true
							w.variables[i] = v
						}
					}
				case 3:
					if w.hourly {
						w.hourlyOpenDate = v
						w.hourlyOpenTime = now
						h := w.lastWriteTime.Add(time.Hour * time.Duration(w.maxHours)).Hour()
						if v == h {
							rotate = true
							w.variables[i] = v
						}
					}
				case 4:
					if w.minutely {
						w.minutelyOpenDate = v
						w.minutelyOpenTime = now
						m := w.lastWriteTime.Add(time.Minute * time.Duration(w.maxMinutes)).Minute()
						if v == m {
							rotate = true
							w.variables[i] = v
						}
					}
				}
			}
		}
	}

	if rotate == false {
		return nil
	}

	w.initFileOnce.Do(w.initFile)
	w.lastWriteTime = now

	if w.fileBufWriter != nil {
		if err := w.fileBufWriter.Flush(); err != nil {
			return err
		}
	}

	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
	}

	filePath := fmt.Sprintf(w.pathFmt, w.variables...)

	if err := os.MkdirAll(path.Dir(filePath), w.rotatePerm); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	if file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, w.rotatePerm); err == nil {
		w.file = file
	} else {
		return err
	}

	if w.fileBufWriter = bufio.NewWriterSize(w.file, 8192); w.fileBufWriter == nil {
		return errors.New("Filewriter new fileBufWriter failed")
	}
	w.suffix = filepath.Ext(filePath)
	w.filenameOnly = strings.TrimSuffix(filePath, w.suffix)
	return nil
}

func getYear(now *time.Time) int {
	return now.Year()
}

func getMonth(now *time.Time) int {
	return int(now.Month())
}

func getDay(now *time.Time) int {
	return now.Day()
}

func getHour(now *time.Time) int {
	return now.Hour()
}

func getMin(now *time.Time) int {
	return now.Minute()
}

func convertPatternToFmt(pattern []byte) string {
	pattern = bytes.Replace(pattern, []byte("%Y"), []byte("%d"), -1)
	pattern = bytes.Replace(pattern, []byte("%M"), []byte("%02d"), -1)
	pattern = bytes.Replace(pattern, []byte("%D"), []byte("%02d"), -1)
	pattern = bytes.Replace(pattern, []byte("%H"), []byte("%02d"), -1)
	pattern = bytes.Replace(pattern, []byte("%m"), []byte("%02d"), -1)
	return string(pattern)
}

func init() {
	pathVariableTable = make(map[byte]func(*time.Time) int, 5)
	pathVariableTable['Y'] = getYear
	pathVariableTable['M'] = getMonth
	pathVariableTable['D'] = getDay
	pathVariableTable['H'] = getHour
	pathVariableTable['m'] = getMin
}
