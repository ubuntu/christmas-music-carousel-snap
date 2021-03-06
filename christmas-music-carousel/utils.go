package main

import (
	"flag"
	"io/ioutil"
	"math/rand"
	"path"
	"strings"
	"sync"
	"time"
)

var quitSignalMutex = sync.Mutex{}

// keep a service alive and restart it if needed.
// stop restarting the service if it's failing too quickly many times in a row.
func keepservicealive(f serviceFn, name string, port string, wg *sync.WaitGroup, quit <-chan struct{}) (chan struct{}, <-chan error) {
	err := make(chan error)
	ready := make(chan struct{})
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
				return
			default:
				if n > maxRestart-1 {
					Debug.Printf("%s did fail starting many times, returning an error", name)
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
		musicdir := path.Join(rootdir, "musics")
		files, _ := ioutil.ReadDir(musicdir)
		for _, f := range files {
			musics = append(musics, path.Join(musicdir, f.Name()))
		}

		// shuffling
		for i := range musics {
			j := r.Intn(i + 1)
			musics[i], musics[j] = musics[j], musics[i]
		}

		// add a little bit of bias by swapping in a chosen list the first music to play
		biasm := []string{"12_Days_Of_Christmas.mid", "Carol_Of_The_Bells.mid", "Jingle_Bells.mid", "Let_It_Snow.mid",
			"O_Come_All_Ye_Faithful.mid", "Rudolph_The_Red_Nosed_Reindeer.mid", "Rockin_Around_The_Christmas_Tree.mid",
			"Santa_Claus_Is_Coming_To_Town.mid", "Sleigh_Ride.mid", "What_Child_Is_This.mid"}
	bias:
		for i, title := range musics {
			for _, tchosen := range biasm {
				if strings.HasSuffix(title, tchosen) {
					musics[0], musics[i] = musics[i], musics[0]
					break bias
				}
			}
		}
	}

	Debug.Printf("List of musics to play: %v", musics)
	return musics, nil
}

// signalQuit safely by closing quit channel. However, doing it only once and can be called
// by multiple goroutines
func signalQuit(quit chan struct{}) {
	quitSignalMutex.Lock()
	signalOnce(quit)
	quitSignalMutex.Unlock()
}

// signalOnce once that a channel is closed. This isn't multiple goroutines safe as most of channels are only closed
// by one goroutine (contrary to the quit one above)
func signalOnce(c chan struct{}) {
	select {
	// non blocking: either closed or receiving data
	case _, opened := <-c:
		// close it if it was opened
		if opened {
			close(c)
		}
	// if c is opened a blocks (waiting for something)
	default:
		close(c)
	}
}
