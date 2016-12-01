package main

import (
	"log"
	"sync"
	"time"
)

func play(midiport string, files []string, wg *sync.WaitGroup, quit <-chan interface{}) <-chan error {
	err := make(chan error)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(err)
		// play indefinitly the list of songs
		for {
			for _, f := range files {
				e := aplaymidi(midiport, f)
				if e != nil {
					err <- e
					return
				}
				select {
				case <-quit:
					log.Printf("Quitting player as submitted")
					return
				case <-time.After(time.Millisecond):
				}
			}
		}
	}()

	return err
}

func aplaymidi(midiport string, filename string) error {
	time.Sleep(3)
	return nil
}
