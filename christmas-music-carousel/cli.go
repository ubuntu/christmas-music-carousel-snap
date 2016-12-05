package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
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

func keepservicealive(f serviceFn, name string, port string, wg *sync.WaitGroup, quit <-chan interface{}) (chan interface{}, <-chan error) {
	err := make(chan error)
	ready := make(chan interface{})
	Debug.Printf("Starting %s watcher", name)

	wg.Add(1)
	go func() {
		defer Debug.Printf("%s watcher stopped", name)
		defer wg.Done()
		defer close(err)

		n := 0
		for {
			start := time.Now()
			e := f(port, ready, quit)
			end := time.Now()

			// check for quitting request
			select {
			case <-quit:
				Debug.Printf("Quit %s watcher as requested", name)
				// send a ready signal in case we never sent it on startup. We are the only goroutine accessing it
				// so it's safe to check if closed
				select {
				case _, opened := <-ready:
					if opened {
						close(ready)
					}
				default:
					close(ready)
				}
				return
			default:
				if n > maxRestart-1 {
					Debug.Printf("%s did fail starting many times, returning an error", name)
					// send a ready signal in case we never sent it on startup. We are the only goroutine accessing it
					// so it's safe to check if closed
					select {
					case _, opened := <-ready:
						if opened {
							close(ready)
						}
					default:
						close(ready)
					}
					err <- e
					return
				}
			}

			if end.Sub(start) < time.Duration(10*time.Second) {
				n++
				Debug.Printf("%s failed to start, restart #%d.", name, n)
			} else {
				n = 0
				Debug.Printf("%s failed, but not immediately, reset as first restart.", name)
			}
		}
	}()
	return ready, err
}

func musicToPlay() ([]string, error) {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	// load musics from args
	musics := flag.Args()

	// load from directory then
	if len(musics) == 0 {
		musicdir := os.Getenv("SNAP")
		if musicdir == "" {
			var err error
			if musicdir, err = filepath.Abs(path.Join(filepath.Dir(os.Args[0]), "..")); err != nil {
				return nil, err
			}
		}
		musicdir = path.Join(musicdir, "musics")
		files, _ := ioutil.ReadDir(musicdir)
		for _, f := range files {
			musics = append(musics, path.Join(musicdir, f.Name()))
		}
	}

	// shuffling
	for i := range musics {
		j := r.Intn(i + 1)
		musics[i], musics[j] = musics[j], musics[i]
	}
	Debug.Printf("List of musics to play: %v", musics)
	return musics, nil
}

func func1(port string, ready chan interface{}, quit <-chan interface{}) error {
	cmd := exec.Command("sleep", "3")
	var errbuf bytes.Buffer
	cmd.Stderr = &errbuf
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
			Debug.Println("Forcing func1 to stop")
			cmd.Process.Kill()
		case <-done:
		}
	}()

	e := cmd.Wait()
	if e != nil {
		return fmt.Errorf("%s: %v", errbuf.String(), e)
	}
	return nil
}
