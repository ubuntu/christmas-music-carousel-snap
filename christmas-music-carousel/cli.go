package main

import (
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	mainPort   = "14:0"
	maxRestart = 5
)

type serviceFn func(port string, ready chan interface{}, quit <-chan interface{}) error

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
			// TODO: handle quit better (racy)
			select {
			case _, opened := <-quit:
				if opened {
					close(quit)
				}
			default:
				close(quit)
			}
			etimidity = nil
			break mainloop
		case err := <-eplayer:
			if err != nil {
				Error.Printf("Fatal error in midi player: %v\n", err)
				rc = 1
				// TODO: handle quit better (racy)
				select {
				case _, opened := <-quit:
					if opened {
						close(quit)
					}
				default:
					close(quit)
				}
				eplayer = nil
				break mainloop
			}
			// TODO: handle quit better (racy)
			select {
			case _, opened := <-quit:
				if opened {
					close(quit)
				}
			default:
				close(quit)
			}
			eplayer = nil
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
