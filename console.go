package log4go

import (
	"fmt"
	"os"
)

type colorRecord Record

// brush is a color join function
type brush func(string) string

// newBrush return a fix color Brush
func newBrush(color string) brush {
	pre := "\033["
	reset := "\033[0m"
	return func(text string) string {
		return fmt.Sprintf("%s%s%s%s%s", pre, color, "m", text, reset)
	}
}

var colors = []brush{
	newBrush("1;37"), // Emergency          white
	newBrush("1;36"), // Alert              cyan
	newBrush("1;35"), // Critical           magenta
	newBrush("1;31"), // Error              red
	newBrush("1;33"), // Warning            yellow
	newBrush("1;32"), // Notice             green
	newBrush("1;34"), // Informational      blue
	newBrush("1;44"), // Debug              Background blue
}

func (r *colorRecord) ColorString() string {
	inf := fmt.Sprintf("%s %s %s %s\n", r.time, LevelFlags[r.level], r.file, r.msg)
	return colors[r.level](inf)
}

func (r *colorRecord) String() string {
	inf := ""
	switch r.level {
	case EMERGENCY:
		inf = fmt.Sprintf("\033[36m%s\033[0m [\033[37m%s\033[0m] \033[47;30m%s\033[0m %s\n",
			r.time, LevelFlags[r.level], r.file, r.msg)
	case ALERT:
		inf = fmt.Sprintf("\033[36m%s\033[0m [\033[36m%s\033[0m] \033[47;30m%s\033[0m %s\n",
			r.time, LevelFlags[r.level], r.file, r.msg)
	case CRITICAL:
		inf = fmt.Sprintf("\033[36m%s\033[0m [\033[35m%s\033[0m] \033[47;30m%s\033[0m %s\n",
			r.time, LevelFlags[r.level], r.file, r.msg)
	case ERROR:
		inf = fmt.Sprintf("\033[36m%s\033[0m [\033[31m%s\033[0m] \033[47;30m%s\033[0m %s\n",
			r.time, LevelFlags[r.level], r.file, r.msg)
	case WARNING:
		inf = fmt.Sprintf("\033[36m%s\033[0m [\033[33m%s\033[0m] \033[47;30m%s\033[0m %s\n",
			r.time, LevelFlags[r.level], r.file, r.msg)
	case NOTICE:
		inf = fmt.Sprintf("\033[36m%s\033[0m [\033[32m%s\033[0m] \033[47;30m%s\033[0m %s\n",
			r.time, LevelFlags[r.level], r.file, r.msg)
	case INFO:
		inf = fmt.Sprintf("\033[36m%s\033[0m [\033[34m%s\033[0m] \033[47;30m%s\033[0m %s\n",
			r.time, LevelFlags[r.level], r.file, r.msg)
	case DEBUG:
		inf = fmt.Sprintf("\033[36m%s\033[0m [\033[44m%s\033[0m] \033[47;30m%s\033[0m %s\n",
			r.time, LevelFlags[r.level], r.file, r.msg)
	}

	return inf
}

// ConsoleWriter console writer define
type ConsoleWriter struct {
	level     int
	color     bool
	fullColor bool // line all with color
}

// ConsoleWriterOptions color field options
type ConsoleWriterOptions struct {
	Level     string `json:"level"`
	On        bool   `json:"on"`
	Color     bool   `json:"color"`
	FullColor bool   `json:"fullColor"`
}

// NewConsoleWriter create new console writer
func NewConsoleWriter() *ConsoleWriter {
	return &ConsoleWriter{}
}

// NewConsoleWriterWithOptions create new console writer with level
func NewConsoleWriterWithOptions(options ConsoleWriterOptions) *ConsoleWriter {
	defaultLevel := DEBUG

	if len(options.Level) != 0 {
		defaultLevel = getLevelDefault(options.Level, defaultLevel)
	}

	return &ConsoleWriter{
		level:     defaultLevel,
		color:     options.Color,
		fullColor: options.FullColor,
	}
}

// Write console write
func (w *ConsoleWriter) Write(r *Record) error {
	if r.level > w.level {
		return nil
	}
	if w.color {
		if w.fullColor {
			fmt.Fprint(os.Stdout, ((*colorRecord)(r)).ColorString())
		} else {
			fmt.Fprint(os.Stdout, ((*colorRecord)(r)).String())
		}
	} else {
		fmt.Fprint(os.Stdout, r.String())
	}
	return nil
}

// Init console init without implement
func (w *ConsoleWriter) Init() error {
	return nil
}

// SetColor console output color control
func (w *ConsoleWriter) SetColor(c bool) {
	w.color = c
}

// SetFullColor console output full line color control
func (w *ConsoleWriter) SetFullColor(c bool) {
	w.fullColor = c
}
