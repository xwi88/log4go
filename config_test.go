package log4go

import (
	"testing"
	"time"
)

var (
	logConfig = `{
  "level": "info",
  "fullPath": true,
	
  "file": {
    "level": "debug",
    "filename": "./test/log4go-test-%Y%M%D.log",
    "on": true
  },

  "console": {
    "level": "error",
    "on": true,
    "color": true
  },
	
  "kafka": {
    "level": "ERROR",
    "on": true,
    "bufferSize": 10,
    "debug": true,
	"msg": {
		"serverIp": "127.0.0.1"
	},
    "specifyVersion":true,
    "version":"0.10.0.1",
    "key": "kafka-test",
    "producerTopic": "log4go-kafka-test",
    "producerReturnSuccesses": true,
    "producerTimeout": 1,
    "brokers": ["47.94.201.80:9092"]
  }
}
`
)

func TestConfig(t *testing.T) {
	if err := SetLog([]byte(logConfig)); err != nil {
		panic(err)
	}
	var name = "log4go config test"
	Debug("log4go by %s debug", name)
	Info("log4go by %s info", name)
	Notice("log4go by %s notice", name)
	Warn("log4go by %s warn", name)
	Error("log4go by %s error", name)
	Critical("log4go by %s critical", name)
	Alert("log4go by %s alert", name)
	Emergency("log4go by %s emergency", name)

	time.Sleep(1 * time.Second)
}
