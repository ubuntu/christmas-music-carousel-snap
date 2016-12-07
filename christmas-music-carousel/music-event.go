package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"os"

	"github.com/oleksandr/bonjour"
)

// look for PiGlow on the network and start the music event processor.
func startPiGlowMusicSync(midiPort string, ready chan interface{}, quit <-chan interface{}) error {
	// get the service ip and port
	resolver, err := bonjour.NewResolver(nil)
	if err != nil {
		return fmt.Errorf("failed to initialize mdns resolver for PiGlow: %v\n", err)
	}
	results := make(chan *bonjour.ServiceEntry)
	go func() {
		resolver.Lookup("PiGlowGRPC", "_piglow._tcp", "", results)
	}()

	// we only get the first result and use those
	var m *bonjour.ServiceEntry
	select {
	case m = <-results:
		resolver.Exit <- true
	case <-quit:
		Debug.Println("Quit PiGlow connector watcher as requested")
		return nil
	case <-time.After(5 * time.Second):
		resolver.Exit <- true
		return fmt.Errorf("no PiGlow service found on the network")
	}

	// grab which binary to run (the one in path or from master)
	cmdName := "music-grpc-events"
	masterCmd := filepath.Join(filepath.Dir(os.Args[0]), "..", "music-grpc-events", "bin", "music-grpc-events-master")
	if _, err := os.Stat(masterCmd); err == nil {
		cmdName = masterCmd
	}

	cmd := exec.Command(cmdName, midiPort, fmt.Sprintf("%s:%d", m.AddrIPv4.String(), m.Port))
	var errbuf bytes.Buffer
	cmd.Stderr = &errbuf
	// prevent Ctrl + C and other signals to get sent
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	err = cmd.Start()
	if err != nil {
		return err
	}

	// killer goroutine
	done := make(chan interface{})
	defer close(done)
	go func() {
		select {
		case <-quit:
			Debug.Println("Quit PiGlow connector watcher as requested")
			cmd.Process.Kill()
		case <-done:
		}
	}()

	Debug.Printf("Signaling PiGlow has be found and connected")
	// we only signal it once, if piglow connector fails and restarts, we don't care about the signal
	select {
	case _, opened := <-ready:
		if opened {
			close(ready)
		}
	default:
		close(ready)
	}

	e := cmd.Wait()
	if e != nil {
		return fmt.Errorf("%s: %v", errbuf.String(), e)
	}
	return nil
}
