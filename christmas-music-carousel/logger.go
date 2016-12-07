package main

// very simple logger library, mostly based on https://dave.cheney.net/2015/11/05/lets-talk-about-logging

import (
	"io/ioutil"
	"log"
	"os"
)

var (
	// Debug logging level
	Debug *log.Logger
	// User normal info logging level
	User *log.Logger
	// Error logging level
	Error *log.Logger
)

const (
	normalLogFlags = 0
	debugLogFlags  = log.Ldate | log.Ltime | log.Lshortfile
)

func init() {
	Debug = log.New(ioutil.Discard, "DEBUG: ", normalLogFlags)
	User = log.New(os.Stdout, "", normalLogFlags)
	Error = log.New(os.Stderr, "ERROR: ", normalLogFlags)
}

// EnableDebug prints debug messages with all details.
func EnableDebug() {
	Debug.SetOutput(os.Stderr)
	Debug.SetFlags(debugLogFlags)
	User.SetFlags(debugLogFlags)
	Error.SetFlags(debugLogFlags)
}

// NormalLogging returns to warning and err only logging state
func NormalLogging() {
	Debug.SetOutput(ioutil.Discard)
	Debug.SetFlags(normalLogFlags)
	User.SetFlags(normalLogFlags)
	Error.SetFlags(normalLogFlags)
}
