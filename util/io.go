package util

import (
	"bufio"
	"io"
	"net/http"
	"os"
)

// ReadLines Read file to get lines from it
func ReadLines(filename string) (lines []string, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return
}

func ReadLastLines(filename string, countLines int) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	rollingLines := make([]string, 0, countLines)

	for scanner.Scan() {
		rollingLines = append(rollingLines, scanner.Text())
		if len(rollingLines) > countLines {
			rollingLines = rollingLines[1:] // Maintain only the last countLines lines
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return rollingLines, nil
}

// WriteLines Write line list to file
func WriteLines(filename string, lines []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	return writer.Flush()
}

func AddLines(filename string, lines []string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	return writer.Flush()
}

func IsDirExists(dirpath string) bool {
	if _, err := os.Stat(dirpath); os.IsNotExist(err) {
		return false
	}
	return true
}

func DownloadFile(url string) ([]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return io.ReadAll(response.Body)
}
