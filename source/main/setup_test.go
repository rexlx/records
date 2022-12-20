package main

import (
	"log"
	"os"
	"testing"
)

var testApp Application

func TestMain(m *testing.M) {
	testApp = Application{
		InfoLog:  log.New(os.Stdout, "info ", log.LUTC),
		ErrorLog: log.New(os.Stdout, "error ", log.LUTC),
		Config:   &RuntimeConfig{},
	}
	os.Exit(m.Run())
}
