package main

import (
	"os/exec"
	"sync"
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
				e := aplaymidi(midiport, f, quit)

				// check for quitting request
				select {
				case <-quit:
					Debug.Println("Quit player watcher as requested")
					return
				default:
				}

				if e != nil {
					err <- e
					return
				}
			}
		}
	}()

	return err
}

func aplaymidi(midiport string, filename string, quit <-chan interface{}) error {
	cmd := exec.Command("sleep", "300")
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
