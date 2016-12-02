package main

import (
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// start and connect timidity daemon to port
func startTimidity(port string, ready chan<- interface{}, quit <-chan interface{}) error {
	cmd := exec.Command("timidity", "-Os", "-iA")
	e := cmd.Start()
	if e != nil {
		return e
	}
	// prevent Ctrl + C and other signals to get sent
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	wg := sync.WaitGroup{}

	// killer goroutine
	done := make(chan interface{})
	defer close(done)
	go func() {
		select {
		case <-quit:
			Debug.Println("Forcing timidity to stop")
		case <-done:
		}
		cmd.Process.Kill()
		cmd = nil
	}()

	// we have 2 goroutines which can send to err
	// if we stop the connect goroutine, the timidity .Wait() will try to send there
	err := make(chan error, 1)
	defer close(err)

	// Timitidy process
	wg.Add(1)
	go func() {
		defer wg.Done()
		Debug.Println("Starting timidity")
		err <- cmd.Wait()
		Debug.Println("Timidity stopped")
	}()

	// connector goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		Debug.Println("Starting timidity connector")
		connectTimitidy(port, ready, done, err)
		Debug.Println("Timidity connector ended")
	}()

	e = <-err
	// signal to kill timidity if still running
	if cmd != nil {
		done <- true
	}
	wg.Wait()
	return e
}

// connect timidity to port, send a ready signal once connected
func connectTimitidy(port string, ready chan<- interface{}, done <-chan interface{}, err chan<- error) {
	n := 0
	for {
		_, e := exec.Command("aconnect", "-l").Output()
		if e != nil {
			if n > 4 {
				err <- e
				return
			}
			n++
			time.Sleep(500 * time.Millisecond)
			continue
		}
		Debug.Printf("Signaling timidity is ready")
		close(ready)
		return
	}

}
