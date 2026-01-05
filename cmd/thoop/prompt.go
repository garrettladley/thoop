package main

import (
	"bufio"
	"context"
	"io"
	"strings"
)

func confirm(ctx context.Context, r io.Reader) bool {
	resultCh := make(chan string, 1)

	go func() {
		reader := bufio.NewReader(r)
		response, _ := reader.ReadString('\n')
		resultCh <- response
	}()

	select {
	case <-ctx.Done():
		return false
	case response := <-resultCh:
		response = strings.TrimSpace(strings.ToLower(response))
		return response == "y" || response == "yes"
	}
}
