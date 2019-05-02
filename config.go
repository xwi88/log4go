package log4go

import (
	"encoding/json"
	"io/ioutil"
)

// LogConfig log config
type LogConfig struct {
	Level    string               `json:"level"`
	FullPath bool                 `json:"fullPath"`
	File     FileWriterOptions    `json:"file"`
	Console  ConsoleWriterOptions `json:"console"`
	KafKa    KafKaWriterOptions   `json:"kafka"`
}

// SetupLog setup log
func SetupLog(lc LogConfig) (err error) {
	defaultLevel := getLevel(lc.Level)
	fullPath := lc.FullPath
	WithFullPath(fullPath)

	if lc.File.On {
		w := NewFileWriterWithOptions(lc.File)
		w.level = getLevelDefault(lc.File.Level, defaultLevel)
		Register(w)
	}

	if lc.Console.On {
		w := NewConsoleWriterWithOptions(lc.Console)
		w.level = getLevelDefault(lc.Console.Level, defaultLevel)
		Register(w)
	}

	if lc.KafKa.On {
		w := NewKafKaWriter(lc.KafKa)
		w.level = getLevelDefault(lc.KafKa.Level, defaultLevel)
		Register(w)
	}
	return nil
}

// SetLogWithConf setup log with config file
func SetLogWithConf(file string) (err error) {
	var lc LogConfig
	cnt, err := ioutil.ReadFile(file)

	if err = json.Unmarshal(cnt, &lc); err != nil {
		return
	}
	return SetupLog(lc)
}

// SetLog setup log with config []byte
func SetLog(config []byte) (err error) {
	var lc LogConfig
	if err = json.Unmarshal(config, &lc); err != nil {
		return
	}
	return SetupLog(lc)
}

func getLevel(flag string) int {
	return getLevelDefault(flag, DEBUG)
}
