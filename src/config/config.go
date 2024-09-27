package config

import (
	"io"
	"portForwarder/src/parser"
	"reflect"
	"strings"
)

func ParseConfig(configReader io.Reader, out interface{}) {
	// Parse the file
	p := parser.NewKeyValueParser(configReader)
	result := p.Parse()

	// use struct tags to map the key-value pairs to the struct fields

	val := reflect.ValueOf(out)

	if val.Kind() != reflect.Ptr {
		panic("out must be a pointer")
	}

	val = val.Elem()
	typ := val.Type()

	for _, r := range result {
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			key := field.Tag.Get("config")
			if key == "" {
				// try to use the field name
				key = strings.ToLower(field.Name)
			}
			value, ok := r[key]
			if ok {
				val.Field(i).SetString(value)
			}
		}
	}
}
