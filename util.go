package GoCryptoTCP

import "log"

func CheckErr(err error) {
	if err != nil {
		log.Println(err)
	}
}

func CheckFatalErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
