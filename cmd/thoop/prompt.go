package main

import (
	"bufio"
	"io"
	"strings"
)

func confirm(r io.Reader) bool {
	reader := bufio.NewReader(r)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
