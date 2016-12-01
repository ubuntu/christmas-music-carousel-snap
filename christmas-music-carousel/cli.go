package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"syscall"
	"time"
)

const mainport = "14:0"

type serviceFn func(quit <-chan interface{}) error

func main() {

	// alsa operations
	if os.Getenv("SUDO_UID") == "" {
		log.Fatalln("This program needs to run as root, under sudo to get access to alsa from the snap")
	}
	// bindmount in current snap namespace /usr/share and /usr/lib directory for alsa conf and plugin not being relocatable
	if os.Getenv("SNAP") != "" {
		if err := syscall.Mount("/var/lib/snapd/hostfs/usr/lib", "/usr/lib", "", syscall.MS_BIND, ""); err == nil {
			log.Fatalf("Couldn't mount alsa directory: %v", err)
		}
		if err := syscall.Mount("/var/lib/snapd/hostfs/usr/share", "/usr/share", "", syscall.MS_BIND, ""); err == nil {
			log.Fatalf("Couldn't mount alsa directory: %v", err)
		}
	}

	// Drop priviledges?

	musics := []string{"Foo", "Bar", "Baz", "Tralala"}

	wg := &sync.WaitGroup{}
	rc := 0
	quit := make(chan interface{})

	// run listen to music event client
	etimidity := keepservicealive(func1, wg, quit)

	// run timidity

	// connect timitidy to main input

	// grab musics to play

	// run aplay with one music at a time
	eplayer := play(mainport, musics, wg, quit)

	fmt.Println("All service started")
mainloop:
	for {
		select {
		case err := <-etimidity:
			fmt.Printf("Fatal error in midi timidity backend player: %v\n", err)
			close(quit)
			rc = 1
			break mainloop
		case err := <-eplayer:
			fmt.Printf("Fatal error in midi player: %v\n", err)
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
				log.Printf("Failed a service quickly, increasing number of quick restarts: %d.", nrestarts)
			} else {
				nrestarts = 1
				log.Printf("Failed a service but after a long time, considering as first restart.")
			}

			select {
			case <-quit:
				log.Printf("Quit submitted")
				return
			case <-time.After(time.Millisecond):
				if nrestarts > 5 {
					log.Printf("We did fail a service many times, returning an error")
					err <- e
					return
				}
			}

		}
	}()
	return err
}

func func1(quit <-chan interface{}) error {
	time.Sleep(time.Second * 3)
	return nil
}
