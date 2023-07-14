package main

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
)

func convertMap(originalMap map[interface{}]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{})

	for key, value := range originalMap {
		stringKey, ok := key.(string)
		if !ok {
			// If the key is not a string, convert it to string using reflection
			stringKey = fmt.Sprintf("%v", key)
		}

		switch value := value.(type) {
		case map[interface{}]interface{}:
			// If the value is a nested map, recursively convert it
			newMap[stringKey] = convertMap(value)
		default:
			// Otherwise, copy the value as-is
			newMap[stringKey] = value
		}
	}

	return newMap
}

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

func main() {
	originalMap := map[interface{}]interface{}{
		"key1": "value1",
		42:     "value2",
		"key3": map[interface{}]interface{}{
			"nestedKey": "nestedValue",
		},
	}

	newMap := convertMap(originalMap)
	fmt.Println(newMap)

	j, err := json.Marshal(newMap)
	if err != nil {
		e := fmt.Errorf("error creating message for REDIS: %w", err)
		fmt.Printf("\nfailed to marshal json with error:%v\n", e.Error())
	} else {
		fmt.Printf("\njson:\n%v\n", string(j))
	}
}
