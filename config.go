package log4go

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

// GlobalLevel global level
var GlobalLevel = DEBUG

// LogConfig log config
type LogConfig struct {
	Level         string               `json:"level" mapstructure:"level"`
	FullPath      bool                 `json:"full_path" mapstructure:"full_path"`
	FileWriter    FileWriterOptions    `json:"file_writer" mapstructure:"file_writer"`
	ConsoleWriter ConsoleWriterOptions `json:"console_writer" mapstructure:"console_writer"`
	KafKaWriter   KafKaWriterOptions   `json:"kafka_writer" mapstructure:"kafka_writer"`
}

// SetupLog setup log
func SetupLog(lc LogConfig) (err error) {
	// global config
	GlobalLevel = getLevel(lc.Level)

	// writer enable
	// 1. if not set level, use global level;
	// 2. if set level, use min level
	validGlobalMinLevel := EMERGENCY // default max level
	validGlobalMinLevelBy := "global"

	if lc.FileWriter.Enable {
		validGlobalMinLevel = maxInt(getLevelDefault(lc.FileWriter.Level, GlobalLevel, "file_writer"), validGlobalMinLevel)
		if validGlobalMinLevel == getLevelDefault(lc.FileWriter.Level, GlobalLevel, "file_writer") {
			validGlobalMinLevelBy = "file_writer"
		}
	}

	if lc.ConsoleWriter.Enable {
		validGlobalMinLevel = maxInt(getLevelDefault(lc.ConsoleWriter.Level, GlobalLevel, "console_writer"), validGlobalMinLevel)
		if validGlobalMinLevel == getLevelDefault(lc.ConsoleWriter.Level, GlobalLevel, "console_writer") {
			validGlobalMinLevelBy = "console_writer"
		}
	}

	if lc.KafKaWriter.Enable {
		validGlobalMinLevel = maxInt(getLevelDefault(lc.KafKaWriter.Level, GlobalLevel, "kafka_writer"), validGlobalMinLevel)
		if validGlobalMinLevel == getLevelDefault(lc.KafKaWriter.Level, GlobalLevel, "kafka_writer") {
			validGlobalMinLevelBy = "kafka_writer"
		}
	}

	fullPath := lc.FullPath
	WithFullPath(fullPath)
	SetLevel(validGlobalMinLevel)

	if lc.FileWriter.Enable {
		w := NewFileWriterWithOptions(lc.FileWriter)
		w.level = getLevelDefault(lc.FileWriter.Level, GlobalLevel, "file_writer")
		Register(w)
	}

	if lc.ConsoleWriter.Enable {
		w := NewConsoleWriterWithOptions(lc.ConsoleWriter)
		w.level = getLevelDefault(lc.ConsoleWriter.Level, GlobalLevel, "console_writer")
		Register(w)
	}

	if lc.KafKaWriter.Enable {
		w := NewKafKaWriter(lc.KafKaWriter)
		w.level = getLevelDefault(lc.KafKaWriter.Level, GlobalLevel, "kafka_writer")
		Register(w)
	}

	log.Printf("log4go validGlobalLevel(min:%v, flag:%v, by:%v, default:%v)",
		validGlobalMinLevel, LevelFlags[validGlobalMinLevel], validGlobalMinLevelBy, LevelFlags[GlobalLevel])
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
	return getLevelDefault(flag, DEBUG, "")
}

// maxInt return max int
func maxInt(a, b int) int {
	if a < b {
		return b
	}
	return a
}
