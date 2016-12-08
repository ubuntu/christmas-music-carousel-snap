package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"
)

const (
	mainPort   = "14:0"
	maxRestart = 5
)

type serviceFn func(port string, ready chan interface{}, quit <-chan interface{}) error

const usageText = `Usage: %s [-options] [LIST OF MIDI FILES]

Play a music carousel and optionally sync up with lights on a Raspberry PiGlow
connected on the network.

A list of midi files can be provided, and in that case, the carousel will play
over them in random orders. If none is provided, a default christmas selection
is chosen.
If you have a PiGlow on the same network, ensure you have the grpc-piglow snap
installed on it.

This programs need to be ran as root on your laptop to connect to alsa.
`

func main() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usageText, path.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	d := flag.Bool("debug", false, "Enable debug messages")
	b := flag.Int("brightness", 0, "Adjust brightness (from 1 to 255) for light up PiGlow. Warning: any value above default (20) is dazzling")
	flag.Parse()

	if *d {
		EnableDebug()
		Debug.Println("Debug message level enabled")
	}
	if *b > 0 {
		setBrightness(*b)
	}

	// alsa operations
	// bindmount in current snap namespace /usr/share and /usr/lib directory for alsa conf and plugin not being relocatable
	// TODO: extract in a function returning err
	if os.Getenv("SNAP") != "" {
		if os.Getenv("SUDO_UID") == "" {
			Error.Println("This program needs to run as root, under sudo to get access to alsa from the snap")
			os.Exit(1)
		}
		if err := syscall.Mount("/var/lib/snapd/hostfs/usr/lib", "/usr/lib", "", syscall.MS_BIND, ""); err != nil {
			Error.Printf("Couldn't mount alsa directory: %v", err)
			os.Exit(1)
		}
		if err := syscall.Mount("/var/lib/snapd/hostfs/usr/share", "/usr/share", "", syscall.MS_BIND, ""); err != nil {
			Error.Printf("Couldn't mount alsa directory: %v", err)
			os.Exit(1)
		}
		// TODO: Drop priviledges?
	}

	wg := &sync.WaitGroup{}
	rc := 0
	quit := make(chan interface{})

	// handle Ctrl + Ctrl properly
	userstop := make(chan os.Signal)
	signal.Notify(userstop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-userstop:
			Debug.Printf("Exit requested")
			close(quit)
		}
	}()

	// run listen to music event client
	pgready, epg := keepservicealive(startPiGlowMusicSync, "Piglow Connector", mainPort, wg, quit)

	// run timidity and connect to main input
	timitidyready, etimidity := keepservicealive(startTimidity, "Timidity", mainPort, wg, quit)

	// grab musics to play and shuffle them
	flag.Parse()
	musics, err := musicToPlay()
	if err != nil {
		Error.Println(err)
		close(quit)
	}

	// give an additional second for the piglow connection to be setup
	select {
	case <-pgready:
	case <-time.After(time.Second):
		User.Printf("Couldn't quickly find a PiGlow on the network, ignoring this feature, but still trying to reconnect")
	}

	// run aplaymidi forever in a loop once timidity is ready
	<-timitidyready
	eplayer := playforever(mainPort, musics, wg, quit)

	Debug.Println("All services started")
mainloop:
	for {
		select {
		case err := <-etimidity:
			Error.Printf("Fatal error in midi timidity backend player: %v\n", err)
			rc = 1
			signalQuit(quit)
			break mainloop
		case err := <-eplayer:
			if err != nil {
				Error.Printf("Fatal error in midi player: %v\n", err)
				rc = 1
			}
			signalQuit(quit)
			break mainloop
		// FIXME: there is a race if 2 errors happens in the same time. Indeed, only one is read and the other
		// goroutine is blocked, not releasing the wait group lock then.
		case <-epg:
			// TODO: separate no PiGlow detected from detected, but an error happened
			User.Println("No working PiGlow detected, continuing without led synchronization support")
			epg = nil
		case <-quit:
			break mainloop
		}
	}

	wg.Wait()
	os.Exit(rc)
}
