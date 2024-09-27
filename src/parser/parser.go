package parser

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

type KeyValue map[string]string

type KeyValueParser struct {
	inputStream io.Reader
}

func NewKeyValueParser(inputStream io.Reader) *KeyValueParser {
	return &KeyValueParser{inputStream: inputStream}
}

func (p *KeyValueParser) Parse() []KeyValue {

	// Read the input stream line by line
	scanner := bufio.NewScanner(p.inputStream)

	var result []KeyValue
	for scanner.Scan() {
		// Use regex to capture key-value pairs
		re := regexp.MustCompile(`(\w+)=(".*?"|\S+)`)
		matches := re.FindAllStringSubmatch(scanner.Text(), -1)

		// Initialize a map to store key-value pairs
		oneLineResult := make(KeyValue)
		for _, match := range matches {
			key := match[1]
			value := strings.Trim(match[2], `"`)

			// Add the key-value pair to the map
			oneLineResult[key] = value
		}

		// Append the map to the result
		result = append(result, oneLineResult)
	}

	return result
}
