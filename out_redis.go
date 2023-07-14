package main

import (
	"C"
	//	"encoding/json"
	"fmt"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
	jsoniter "github.com/json-iterator/go"
	zerolog "github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"

	"os"
	"time"
)

var (
	rc   *redisClient
	json = jsoniter.ConfigCompatibleWithStandardLibrary
	// both variables are set in Makefile
	revision  string
	builddate string
	plugin    Plugin = &fluentPlugin{}
)

//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	return output.FLBPluginRegister(ctx, "redis", "Redis Output Plugin.")
}

type logmessage struct {
	data []byte
}

type Plugin interface {
	Environment(ctx unsafe.Pointer, key string) string
	Unregister(ctx unsafe.Pointer)
	GetRecord(dec *output.FLBDecoder) (ret int, ts interface{}, rec map[interface{}]interface{})
	NewDecoder(data unsafe.Pointer, length int) *output.FLBDecoder
	Send(values []*logmessage) error
	Exit(code int)
}

type fluentPlugin struct{}

func (p *fluentPlugin) Environment(ctx unsafe.Pointer, key string) string {
	return output.FLBPluginConfigKey(ctx, key)
}

func (p *fluentPlugin) Unregister(ctx unsafe.Pointer) {
	output.FLBPluginUnregister(ctx)
}

func (p *fluentPlugin) GetRecord(dec *output.FLBDecoder) (int, interface{}, map[interface{}]interface{}) {
	return output.GetRecord(dec)
}

func (p *fluentPlugin) NewDecoder(data unsafe.Pointer, length int) *output.FLBDecoder {
	return output.NewDecoder(data, int(length))
}

func (p *fluentPlugin) Exit(code int) {
	os.Exit(code)
}

func (p *fluentPlugin) Send(values []*logmessage) error {
	return rc.send(values)
}

// ctx (context) pointer to fluentbit context (state/ c code)
//
//export FLBPluginInit
func FLBPluginInit(ctx unsafe.Pointer) int {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	hosts := plugin.Environment(ctx, "Hosts")
	password := plugin.Environment(ctx, "Password")
	key := plugin.Environment(ctx, "Key")
	db := plugin.Environment(ctx, "DB")
	usetls := plugin.Environment(ctx, "UseTLS")
	tlsskipverify := plugin.Environment(ctx, "TLSSkipVerify")

	// create a pool of redis connection pools
	config, err := getRedisConfig(hosts, password, db, usetls, tlsskipverify, key)
	if err != nil {
		//fmt.Printf("configuration errors: %v\n", err)
		log.Error().Str("app", "out-redis").Str("build", builddate).Str("rev", revision).
			Err(err).Msgf("failed config with value: %v", config)
		// FIXME use fluent-bit method to err in init
		plugin.Unregister(ctx)
		plugin.Exit(1)
		return output.FLB_ERROR
	}
	rc = &redisClient{
		pools: newPoolsFromConfig(config),
		key:   config.key,
	}
	//fmt.Printf("[out-redis] build:%s version:%s redis connection: %s\n", builddate, revision, config)
	log.Info().Str("app", "out-redis").Str("build", builddate).Str("rev", revision).
		Msgf("succeed to connect to redis with config %s", config)
	return output.FLB_OK
}

// FLBPluginFlush is called from fluent-bit when data need to be sent. is called from fluent-bit when data need to be sent.
//
//export FLBPluginFlush
func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
	var ret int
	var ts interface{}
	var record map[interface{}]interface{}

	// Create Fluent Bit decoder
	dec := plugin.NewDecoder(data, int(length))

	// Iterate Records

	var logs []*logmessage

	for {
		// Extract Record
		ret, ts, record = plugin.GetRecord(dec)
		if ret != 0 {
			break
		}

		// Print record keys and values
		var timeStamp time.Time
		switch t := ts.(type) {
		case output.FLBTime:
			timeStamp = ts.(output.FLBTime).Time
		case uint64:
			timeStamp = time.Unix(int64(t), 0)
		case []interface{}:
			//var e error
			//s := make([]byte, len(t))
			//for i, v := range t {
			//	s[i] = v.(output.FLBTime)
			//}
			//timeStamp = s[0].(output.FLBTime).Time
			//timeStamp, e = time.Parse(time.RFC3339, string(s))
			//if e != nil {
			//	fmt.Printf("given time %v is not in a known format %T, defaulting to now.\n", ts, t)
			//	timeStamp = time.Now()
			//}
			//fmt.Printf("array given time %v is not in a known format %T, defaulting to now.\n", ts, t)
			timeStamp = t[0].(output.FLBTime).Time
		default:
			//fmt.Printf("given time %v is not in a known format %T, defaulting to now.\n", ts, t)
			log.Info().Str("app", "out-redis").Str("build", builddate).Str("rev", revision).
				Msgf("timestamp %v with %T in record is not in a supported type, so set to default of now", ts, t)
			timeStamp = time.Now()
		}

		//fmt.Printf("\nflush.record.%v:\n%v\n", record, timeStamp)

		//j, err := json.Marshal(record)
		//if err != nil {
		//	e := fmt.Errorf("error creating message for REDIS: %w", err)
		//	fmt.Printf("\nflush.record.json:\nfailed with error:%v\n", e.Error())
		//} else {
		//	fmt.Printf("\nflush.record.json:\n%v\n", string(j))
		//}

		// fmt.Printf("\nconvertMap.begin:\n")
		// recordNew := convertMap(record)
		// fmt.Printf("\nconvertMap.end:\n")
		// //fmt.Println(recordNew)
		// j, err := json.Marshal(recordNew)
		// if err != nil {
		// 	e := fmt.Errorf("error creating message for REDIS: %w", err)
		// 	fmt.Printf("\nfailed to marshal json with error:%v\n", e.Error())
		// } else {
		// 	fmt.Printf("\njson:\n%v\n", string(j))
		// }

		js, err := createJSON(timeStamp, C.GoString(tag), record)
		if err != nil {
			//fmt.Printf("%v\n", err)
			log.Error().Str("app", "out-redis").Str("build", builddate).Str("rev", revision).
				Err(err).Msg("failed to create json")
			// DO NOT RETURN HERE becase one message has an error when json is
			// generated, but a retry would fetch ALL messages again. instead an
			// error should be printed to console
			continue
		}
		logs = append(logs, js)
	}

	err := plugin.Send(logs)
	if err != nil {
		//fmt.Printf("%v\n", err)
		log.Error().Str("app", "out-redis").Str("build", builddate).Str("rev", revision).
			Err(err).Msg("failed to send log")
		return output.FLB_RETRY
	}

	//fmt.Printf("pushed %d logs\n", len(logs))
	log.Info().Str("app", "out-redis").Str("build", builddate).Str("rev", revision).
		Msgf("succeed to push %d logs", len(logs))

	// Return options:
	//
	// output.FLB_OK    = data have been processed.
	// output.FLB_ERROR = unrecoverable error, do not try this again.
	// output.FLB_RETRY = retry to flush later.
	return output.FLB_OK
}

func parseMap(iMap map[interface{}]interface{}) map[string]interface{} {
	//fmt.Printf("\nparse.map:\n%v\n", iMap)

	var err error
	m := make(map[string]interface{})
	for k, v := range iMap {
		switch v := v.(type) {
		case []byte:
			// prevent encoding to base64
			//fmt.Printf("\nparse.[]byte\nkey: %v, type: %T, value: %v\n", k.(string), v, string(v))
			m[k.(string)] = string(v)
		case []interface{}:
			//fmt.Printf("\nparse.[]interface\nkey: %v, type: %T, value: %v\n", k.(string), v, v)
			m[k.(string)] = parseArray(v)
		case map[interface{}]interface{}:
			m[k.(string)] = parseMap(v)
		default:
			//fmt.Printf("\nparse.default\nkey: %v, type: %T, value: %v\n", k.(string), v, v)
			m[k.(string)] = v
		}

		if err != nil {
			break
		}
	}
	return m
}

func parseArray(iArray []interface{}) []interface{} {
	newArray := make([]interface{}, 0)

	for _, value := range iArray {

		switch value := value.(type) {
		case map[interface{}]interface{}:
			//fmt.Printf("\n\tvalue[%d](%T) %v\n", i, value, value)
			// If the value is a nested map, recursively convert it
			newArray = append(newArray, parseMap(value))
		case []uint8:
			//fmt.Printf("\n\t\tvalue[%d](%T) %v %v", i, value, value, string(value))
			newArray = append(newArray, string(value))
		case []interface{}:
			//fmt.Printf("\n\tvalue[%d](%T) %v\n", i, value, value)
			newArray = append(newArray, parseArray(value))
		default:
			//fmt.Printf("\n\tvalue(%T) %v\n", value, value)
			// Otherwise, copy the value as-is
			newArray = append(newArray, value)
		}
	}

	return newArray
}

func createJSON(timestamp time.Time, tag string, record map[interface{}]interface{}) (*logmessage, error) {
	m := parseMap(record)
	// convert timestamp to RFC3339Nano which is logstash format
	m["@timestamp"] = timestamp.UTC().Format(time.RFC3339Nano)
	m["@tag"] = tag

	js, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("error creating message for REDIS: %w", err)
	} else {
		fmt.Printf("\ncreate.json.ok\n%v\n", string(js))
	}
	return &logmessage{data: js}, nil
}

//export FLBPluginExit
func FLBPluginExit() int {
	rc.pools.closeAll()
	return output.FLB_OK
}

func main() {
}
