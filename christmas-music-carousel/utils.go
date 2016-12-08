package main

import (
	"flag"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

// keep a service alive and restart it if needed.
// stop restarting the service if it's failing too quickly many times in a row.
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

// grab a list of musics to play. Can be local list of musics if no argument is provided.
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

		// shuffling
		for i := range musics {
			j := r.Intn(i + 1)
			musics[i], musics[j] = musics[j], musics[i]
		}
	}

	Debug.Printf("List of musics to play: %v", musics)
	return musics, nil
}
