package main

import (
	"autocom/internal/zlhubcli"
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	app := &zlhubcli.App{
		Out:        os.Stdout,
		Err:        os.Stderr,
		Getenv:     os.Getenv,
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
		Now:        time.Now,
	}
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
