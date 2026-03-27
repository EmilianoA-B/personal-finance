package main

import (
	"os"
)

func writePdf(pdfBytes []byte, name string) {
	file, err := os.OpenFile("./pdf/"+name, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	file.Write(pdfBytes)
}
