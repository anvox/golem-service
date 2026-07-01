package main

import (
	"os"
	"regexp"
	"strings"
)

func readEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	vars := make(map[string]string)
	// We can use a scanner to parse lines
	// For simple key=value pairs
	// Match lines containing '='
	// Support comments '#'
	var r = regexp.MustCompile(`^\s*([^#=\s]+)\s*=\s*(.*)\s*$`)
	var scanner = bufioScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches := r.FindStringSubmatch(line)
		if len(matches) == 3 {
			vars[matches[1]] = strings.Trim(matches[2], `"'`)
		}
	}
	return vars, scanner.Err()
}

func bufioScanner(file *os.File) *bufioMock {
	return &bufioMock{file: file}
}

type bufioMock struct {
	file *os.File
	err  error
	line string
	buf  []byte
}

func (b *bufioMock) Scan() bool {
	// Simple line reader
	var lineBytes []byte
	var char = make([]byte, 1)
	for {
		n, err := b.file.Read(char)
		if err != nil {
			if n > 0 {
				lineBytes = append(lineBytes, char[0])
			}
			if len(lineBytes) > 0 {
				b.line = string(lineBytes)
				return true
			}
			b.err = err
			return false
		}
		if char[0] == '\n' {
			b.line = string(lineBytes)
			return true
		}
		lineBytes = append(lineBytes, char[0])
	}
}

func (b *bufioMock) Text() string {
	return b.line
}

func (b *bufioMock) Err() error {
	if b.err.Error() == "EOF" {
		return nil
	}
	return b.err
}
