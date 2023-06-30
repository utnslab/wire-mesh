package main

import (
	"github.com/buger/jsonparser"
)

func main() {
	jsonparser.Get([]byte(`{"name": "John"}`), "name")
}
