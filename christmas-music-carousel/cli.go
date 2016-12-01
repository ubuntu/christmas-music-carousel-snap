package main

import (
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const mainport = "14:0"

type serviceFn func(quit <-chan interface{}) error

func main() {

	// alsa operations
	// bindmount in current snap namespace /usr/share and /usr/lib directory for alsa conf and plugin not being relocatable
	// TODO: extract in a function returning err
	if os.Getenv("SNAP") != "" {
		if os.Getenv("SUDO_UID") == "" {
			Error.Println("This program needs to run as root, under sudo to get access to alsa from the snap")
			os.Exit(1)
		}
		if err := syscall.Mount("/var/lib/snapd/hostfs/usr/lib", "/usr/lib", "", syscall.MS_BIND, ""); err == nil {
			Error.Printf("Couldn't mount alsa directory: %v", err)
			os.Exit(1)
		}
		if err := syscall.Mount("/var/lib/snapd/hostfs/usr/share", "/usr/share", "", syscall.MS_BIND, ""); err == nil {
			Error.Printf("Couldn't mount alsa directory: %v", err)
			os.Exit(1)
		}
		// TODO: Drop priviledges?
	}

	musics := []string{"Foo", "Bar", "Baz", "Tralala"}

	wg := &sync.WaitGroup{}
	rc := 0
	quit := make(chan interface{})

	// run listen to music event client
	etimidity := keepservicealive(func1, wg, quit)

	// run timidity

	// connect timitidy to main input

	// grab musics to play and shuffle them

	// run aplay with one music at a time
	eplayer := play(mainport, musics, wg, quit)

	Debug.Println("All services started")
mainloop:
	for {
		select {
		case err := <-etimidity:
			Error.Printf("Fatal error in midi timidity backend player: %v\n", err)
			close(quit)
			rc = 1
			break mainloop
		case err := <-eplayer:
			Error.Printf("Fatal error in midi player: %v\n", err)
			close(quit)
			rc = 1
			break mainloop
		}
	}

	wg.Wait()
	os.Exit(rc)
}

func keepservicealive(f serviceFn, wg *sync.WaitGroup, quit <-chan interface{}) <-chan error {
	err := make(chan error)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(err)

		nrestarts := 0
		for {
			start := time.Now()
			e := f(quit)
			end := time.Now()
			if end.Sub(start) < time.Duration(10*time.Second) {
				nrestarts++
				Debug.Printf("Failed a service quickly, increasing number of quick restarts: %d.", nrestarts)
			} else {
				nrestarts = 1
				Debug.Printf("Failed a service but after a long time, considering as first restart.")
			}

			select {
			case <-quit:
				Debug.Printf("Quit watch for service as submitted")
				return
			case <-time.After(time.Millisecond):
				if nrestarts > 5 {
					Debug.Printf("We did fail a service many times, returning an error")
					err <- e
					return
				}
			}
		}
	}()
	return err
}

func func1(quit <-chan interface{}) error {
	cmd := exec.Command("sleep", "3")
	err := cmd.Start()
	if err != nil {
		return err
	}

	// kill goroutine
	done := make(chan interface{})
	defer close(done)
	go func() {
		select {
		case <-quit:
			Debug.Println("Forcing func1 to stop")
			cmd.Process.Kill()
		case <-done:
		}
	}()

	return cmd.Wait()
}
