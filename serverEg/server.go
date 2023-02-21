package main

import (
	"GoCryptoTCP"
)

func main() {
	s := GoCryptoTCP.NewServer("127.0.0.1:7730")
	go s.Scan()
	s.StartListen()
}
