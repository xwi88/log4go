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
	Level    string               `json:"level" mapstructure:"level"`
	FullPath bool                 `json:"fullPath" mapstructure:"full_path"`
	File     FileWriterOptions    `json:"file" mapstructure:"file"`
	Console  ConsoleWriterOptions `json:"console" mapstructure:"console"`
	KafKa    KafKaWriterOptions   `json:"kafka" mapstructure:"kafka"`
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

	if lc.File.Enable {
		validGlobalMinLevel = maxInt(getLevelDefault(lc.File.Level, GlobalLevel, "file_writer"), validGlobalMinLevel)
		if validGlobalMinLevel == getLevelDefault(lc.File.Level, GlobalLevel, "file_writer") {
			validGlobalMinLevelBy = "file_writer"
		}
	}

	if lc.Console.Enable {
		validGlobalMinLevel = maxInt(getLevelDefault(lc.Console.Level, GlobalLevel, "console_writer"), validGlobalMinLevel)
		if validGlobalMinLevel == getLevelDefault(lc.Console.Level, GlobalLevel, "console_writer") {
			validGlobalMinLevelBy = "console_writer"
		}
	}

	if lc.KafKa.Enable {
		validGlobalMinLevel = maxInt(getLevelDefault(lc.KafKa.Level, GlobalLevel, "kafka_writer"), validGlobalMinLevel)
		if validGlobalMinLevel == getLevelDefault(lc.KafKa.Level, GlobalLevel, "kafka_writer") {
			validGlobalMinLevelBy = "kafka_writer"
		}
	}

	fullPath := lc.FullPath
	WithFullPath(fullPath)
	SetLevel(validGlobalMinLevel)

	if lc.File.Enable {
		w := NewFileWriterWithOptions(lc.File)
		w.level = getLevelDefault(lc.File.Level, GlobalLevel, "file_writer")
		Register(w)
	}

	if lc.Console.Enable {
		w := NewConsoleWriterWithOptions(lc.Console)
		w.level = getLevelDefault(lc.Console.Level, GlobalLevel, "console_writer")
		Register(w)
	}

	if lc.KafKa.Enable {
		w := NewKafKaWriter(lc.KafKa)
		w.level = getLevelDefault(lc.KafKa.Level, GlobalLevel, "kafka_writer")
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
