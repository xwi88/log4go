package log4go

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/Shopify/sarama"
)

// KafKaMSGFields kafka msg fields
type KafKaMSGFields struct {
	Level     string // dynamic, set by logger, mark the record level
	File      string `json:"file"`      // source code file:line_number
	Message   string `json:"message"`   // required, dynamic
	ServerIP  string `json:"serverIp"`  // required, init field, set by app
	Timestamp string `json:"timestamp"` // required, dynamic, set by logger
	Now       int64  `json:"now"`       // choice

	ExtraFields map[string]interface{} `json:"extraFields"` // extra fields will be added
}

// KafKaWriterOptions kafka writer options
type KafKaWriterOptions struct {
	Level          string `json:"level"`
	On             bool   `json:"on"`
	BufferSize     int    `json:"bufferSize"`
	Debug          bool   `json:"debug"`          // if true, will output the send msg
	SpecifyVersion bool   `json:"specifyVersion"` // if use the input version, default false
	VersionStr     string `json:"version"`        // used to specify the kafka version, ex: 0.10.0.1 or 1.1.1

	Key string `json:"key"` // kafka producer key, choice field

	ProducerTopic           string        `json:"producerTopic"`
	ProducerReturnSuccesses bool          `json:"producerReturnSuccesses"`
	ProducerTimeout         time.Duration `json:"producerTimeout"`
	Brokers                 []string      `json:"brokers"`

	MSG KafKaMSGFields `json:"msg"`
}

// KafKaWriter kafka writer
type KafKaWriter struct {
	level    int
	producer sarama.SyncProducer
	messages chan *sarama.ProducerMessage
	options  KafKaWriterOptions

	run  bool // avoid the block with no running kafka writer
	quit chan struct{}
	stop chan struct{}
}

// NewKafKaWriter new kafka writer
func NewKafKaWriter(options KafKaWriterOptions) *KafKaWriter {
	defaultLevel := DEBUG
	if len(options.Level) != 0 {
		defaultLevel = getLevelDefault(options.Level, defaultLevel)
	}

	if options.Debug {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	return &KafKaWriter{
		options: options,
		quit:    make(chan struct{}),
		stop:    make(chan struct{}),
		level:   defaultLevel,
	}
}

// Init service for Record
func (k *KafKaWriter) Init() error {
	return k.Start()
}

// Write service for Record
func (k *KafKaWriter) Write(r *Record) error {
	if r.level > k.level {
		return nil
	}

	logMsg := r.msg
	if logMsg == "" {
		return nil
	}
	data := k.options.MSG
	// timestamp, level
	data.Level = LevelFlags[r.level]
	now := time.Now()
	data.Now = now.Unix()
	data.Timestamp = now.Format(timestampLayout)
	data.Message = logMsg
	data.File = r.file

	byteData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	var structData map[string]interface{}
	err = json.Unmarshal(byteData, &structData)
	if err != nil {
		log.Printf("[log4go] kafka writer err: %v", err.Error())
	}
	delete(structData, "extraFields")

	// not exist new fields will be added
	for k, v := range data.ExtraFields {
		if _, ok := structData[k]; !ok {
			structData[k] = v
		}
	}
	structDataByte, err := json.Marshal(structData)
	if err != nil {
		return err
	}

	jsonData := string(structDataByte)

	key := ""
	if k.options.Key != "" {
		key = k.options.Key
	}

	msg := &sarama.ProducerMessage{
		Topic: k.options.ProducerTopic,
		// autofill or use specify timestamp, you must set Version >= sarama.V0_10_0_1
		// Timestamp: time.Now(),
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(jsonData),
	}

	if k.options.Debug {
		log.Printf("[log4go] msg [topic: %v, timestamp: %v, brokers: %v]\nkey:   %v\nvalue: %v\n", msg.Topic,
			msg.Timestamp, k.options.Brokers, key, jsonData)
	}
	go k.asyncWriteMessages(msg)

	return nil
}

func (k *KafKaWriter) asyncWriteMessages(msg *sarama.ProducerMessage) {
	if msg != nil {
		k.messages <- msg
	}
}

// send kafka message to kafka
func (k *KafKaWriter) daemonProducer() {
	k.run = true

next:
	for {
		select {
		case mes, ok := <-k.messages:
			if ok {
				partition, offset, err := k.producer.SendMessage(mes)

				if err != nil {
					log.Printf("[log4go] SendMessage(topic=%s, partition=%v, offset=%v, key=%s, value=%s,timstamp=%v) err=%s\n\n", mes.Topic,
						partition, offset, mes.Key, mes.Value, mes.Timestamp, err.Error())
					continue
				} else {
					if k.options.Debug {
						log.Printf("[log4go] SendMessage(topic=%s, partition=%v, offset=%v, key=%s, value=%s,timstamp=%v)\n\n", mes.Topic,
							partition, offset, mes.Key, mes.Value, mes.Timestamp)
					}
				}
			}
		case <-k.stop:
			break next
		}
	}
	k.quit <- struct{}{}
}

// Start start the kafka writer
func (k *KafKaWriter) Start() (err error) {
	log.Printf("[log4go] start kafka writer ...")
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = k.options.ProducerReturnSuccesses
	cfg.Producer.Timeout = k.options.ProducerTimeout

	// if want set timestamp for data should set version
	versionStr := k.options.VersionStr
	kafkaVer := sarama.V0_10_0_1

	if k.options.SpecifyVersion {
		if versionStr != "" {
			if kafkaVersion, err := sarama.ParseKafkaVersion(versionStr); err == nil {
				// should be careful set the version, maybe occur EOF error
				kafkaVer = kafkaVersion
			}
		}
	}
	// if not specify the version, use the sarama.V0_10_0_1 to guarante the timestamp can be control
	cfg.Version = kafkaVer

	// NewHashPartitioner returns a Partitioner which behaves as follows. If the message's key is nil then a
	// random partition is chosen. Otherwise the FNV-1a hash of the encoded bytes of the message key is used,
	// modulus the number of partitions. This ensures that messages with the same key always end up on the
	// same partition.
	// cfg.Producer.Partitioner = sarama.NewHashPartitioner
	// cfg.Producer.Partitioner = sarama.NewRandomPartitioner
	cfg.Producer.Partitioner = sarama.NewRoundRobinPartitioner
	// cfg.Producer.Partitioner = sarama.NewReferenceHashPartitioner

	k.producer, err = sarama.NewSyncProducer(k.options.Brokers, cfg)
	if err != nil {
		log.Printf("[log4go] sarama.NewSyncProducer err, message=%s", err.Error())
		return err
	}
	size := k.options.BufferSize
	if size <= 1 {
		size = 100
	}
	k.messages = make(chan *sarama.ProducerMessage, size)

	go k.daemonProducer()
	log.Printf("[log4go] start kafka writer ok")
	return err
}

// Stop stop the kafka writer
func (k *KafKaWriter) Stop() {
	if k.run {
		close(k.messages)
		<-k.stop
		err := k.producer.Close()
		if err != nil {
			log.Printf("[log4go] kafkaWriter stop error: %v", err.Error())
		}
	}
}
