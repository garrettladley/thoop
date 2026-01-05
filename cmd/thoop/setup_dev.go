//go:build dev

package main

import "github.com/joho/godotenv"

func init() {
	_ = godotenv.Load()
}
