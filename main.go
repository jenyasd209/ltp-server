package main

import (
	"github.com/jenyasd209/ltp-server/src"
)

func main() {
	server := src.NewServer(":8080", src.DefaultPriceRequester())
	err := server.Listen()
	if err != nil {
		panic(err)
	}
}
