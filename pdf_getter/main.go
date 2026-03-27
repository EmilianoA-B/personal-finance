package main

import (
	"log"
	"os"
	"persFinance/auth"

	"github.com/alexflint/go-arg"
)

func main() {

	// "in:inbox from:(clientes@bbva.mx) subject:(Estados de Cuenta BBVA)", "after:2026/3/2", "before:2026/3/17"
	var args struct {
		Filter     string `arg:"positional,required"`
		AfterDate  string `arg:"--after"`
		BeforeDate string `arg:"--before"`
	}
	arg.MustParse(&args)
	if !checkFirstTime() {
		err := auth.GetTokens()
		if err != nil {
			log.Fatalf("Could not aquire token: %v", err)
		}
	}
	err := GetPdfFiles(args.Filter, args.AfterDate, args.BeforeDate)
	if err != nil {
		log.Printf("Error getting PDF files from email:\n %v \n", err)
	}
}

func checkFirstTime() bool {
	currDir, err := os.ReadDir("./secret")
	if err != nil {
		log.Panicf("Error reading directory: %v", err)
	}
	for _, dirEntry := range currDir {
		if dirEntry.Name() == "token.json" {
			return true
		}
	}
	return false
}
