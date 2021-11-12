# log4go

The simple but powerful log for go.

## TODO

- [x] base logger
- [x] console writer
- [x] file writer
- [x] kafka writer
- [ ] net writer

## ENV

The go version shall >= `1.16`

## Writers

- support output the caller`s file and lines
- support level filter
- simply use, pls ref `xxx_test.go`

### FileWriter

>Filename regex support: `%Y` `%M` `%D` `%H` `%m`, prefix must be `%`

### KafkaWriter

>Can writer to kafka easily, with `es_index` you can also transfer data to ES easily. If you want more fields can set
> them by the field `msg.extra_fields`.
