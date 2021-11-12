package log4go

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

// GlobalLevel global level
var GlobalLevel = DEBUG

const (
	WriterNameConsole = "console_writer"
	WriterNameFile    = "file_writer"
	WriterNameKafka   = "kafka_writer"
)

// LogConfig log config
type LogConfig struct {
	Level         string               `json:"level" mapstructure:"level"`
	Debug         bool                 `json:"debug" mapstructure:"debug"` // output log info or not for log4go
	FullPath      bool                 `json:"full_path" mapstructure:"full_path"`
	ConsoleWriter ConsoleWriterOptions `json:"console_writer" mapstructure:"console_writer"`
	FileWriter    FileWriterOptions    `json:"file_writer" mapstructure:"file_writer"`
	KafKaWriter   KafKaWriterOptions   `json:"kafka_writer" mapstructure:"kafka_writer"`
}

// SetupLog setup log
func SetupLog(lc LogConfig) (err error) {
	if !lc.Debug {
		log.SetOutput(ioutil.Discard)
		defer log.SetOutput(os.Stdout)
	}

	// global config
	GlobalLevel = getLevel(lc.Level)

	// writer enable
	// 1. if not set level, use global level;
	// 2. if set level, use min level
	validGlobalMinLevel := EMERGENCY // default max level
	validGlobalMinLevelBy := "global"

	fileWriterDefault := GlobalLevel
	consoleWriterDefault := GlobalLevel
	kafkaWriterDefault := GlobalLevel

	if lc.ConsoleWriter.Enable {
		consoleWriterDefault = getLevelDefault(lc.ConsoleWriter.Level, GlobalLevel, WriterNameConsole)
		validGlobalMinLevel = maxInt(consoleWriterDefault, validGlobalMinLevel)
		if validGlobalMinLevel == consoleWriterDefault {
			validGlobalMinLevelBy = WriterNameConsole
		}
	}

	if lc.FileWriter.Enable {
		fileWriterDefault = getLevelDefault(lc.FileWriter.Level, GlobalLevel, WriterNameFile)
		validGlobalMinLevel = maxInt(fileWriterDefault, validGlobalMinLevel)
		if validGlobalMinLevel == fileWriterDefault {
			validGlobalMinLevelBy = WriterNameFile
		}
	}

	if lc.KafKaWriter.Enable {
		kafkaWriterDefault = getLevelDefault(lc.KafKaWriter.Level, GlobalLevel, WriterNameKafka)
		validGlobalMinLevel = maxInt(kafkaWriterDefault, validGlobalMinLevel)
		if validGlobalMinLevel == kafkaWriterDefault {
			validGlobalMinLevelBy = WriterNameKafka
		}

	}

	fullPath := lc.FullPath
	WithFullPath(fullPath)
	SetLevel(validGlobalMinLevel)

	if lc.ConsoleWriter.Enable {
		w := NewConsoleWriterWithOptions(lc.ConsoleWriter)
		w.level = consoleWriterDefault
		log.Printf("[log4go] enable " + WriterNameConsole + " with level " + LevelFlags[consoleWriterDefault])
		Register(w)
	}

	if lc.FileWriter.Enable {
		w := NewFileWriterWithOptions(lc.FileWriter)
		w.level = fileWriterDefault
		log.Printf("[log4go] enable    " + WriterNameFile + " with level " + LevelFlags[fileWriterDefault])
		Register(w)
	}

	if lc.KafKaWriter.Enable {
		w := NewKafKaWriter(lc.KafKaWriter)
		w.level = kafkaWriterDefault
		log.Printf("[log4go] enable   " + WriterNameKafka + " with level " + LevelFlags[kafkaWriterDefault])
		Register(w)
	}

	log.Printf("[log4go] valid global_level(min:%v, flag:%v, by:%v), default(%v, flag:%v)",
		validGlobalMinLevel, LevelFlags[validGlobalMinLevel], validGlobalMinLevelBy, GlobalLevel, LevelFlags[GlobalLevel])
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
