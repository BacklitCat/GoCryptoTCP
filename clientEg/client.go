package main

import (
	"GoCryptoTCP"
	"log"
)

func main() {
	client := GoCryptoTCP.NewCryptoClient("127.0.0.1:7730")
	_ = client.Dial()
	log.Printf("\n\n  Usage:\n" +
		"    status - show connection status\n" +
		"    to [id] [msg] - send msg to user, eg: to 1000 hello\n\n")
	go client.Scan()
	client.HandleConn()
}
