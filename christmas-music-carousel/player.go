package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

func playforever(midiport string, files []string, wg *sync.WaitGroup, quit <-chan interface{}) <-chan error {
	err := make(chan error)

	wg.Add(1)
	go func() {
		defer Debug.Println("Player watcher stopped")
		defer wg.Done()
		defer close(err)

		// play indefinitly the list of songs
		for {
			var lasterror error
			readOneMusic := false
			for _, f := range files {
				start := time.Now()
				lasterror = aplaymidi(midiport, f, quit)
				end := time.Now()

				// check for quitting request
				select {
				case <-quit:
					Debug.Println("Quit player watcher as requested")
					return
				default:
				}

				if end.Sub(start) > time.Duration(time.Second) {
					readOneMusic = true
				}
			}

			// exit loop if we couldn't play any music
			if !readOneMusic {
				if lasterror != nil {
					err <- lasterror
					return
				}
				err <- errors.New("aplaymidi fails playing any files")
				return
			}
		}
	}()

	return err
}

func aplaymidi(midiport string, filename string, quit <-chan interface{}) error {
	cmd := exec.Command("sleep", "10")
	// prevent Ctrl + C and other signals to get sent
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	err := cmd.Start()
	if err != nil {
		return err
	}

	// killer goroutine
	done := make(chan interface{})
	defer close(done)
	go func() {
		select {
		case <-quit:
			Debug.Println("Forcing aplaymidi to stop")
			cmd.Process.Kill()
		case <-done:
		}
	}()

	return cmd.Wait()
}
