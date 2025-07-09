package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"

	fsevents "github.com/tywkeene/go-fsevents"
)

func handleEvents(w *fsevents.Watcher, quit chan bool) error {
	// Watch for events
	go w.Watch()
	for {
		select {
		case event := <-w.Events:
			if !event.IsDirEvent() {
				// log.Printf("Path: \"%s\"    Mask: %d", event.Path, event.RawEvent.Mask)
				if fsevents.CheckMask(fsevents.Open, event.RawEvent.Mask) {
					// Register the file
					FsEvents.Set(event.Path)
				} else if fsevents.CheckMask(fsevents.CloseWrite, event.RawEvent.Mask) {
					// Calculate elapsed recover time
					FsEvents.CloseWr((event.Path))
				} else if fsevents.CheckMask(fsevents.CloseRead, event.RawEvent.Mask) {
					// Calculate elapsed time
					FsEvents.Upd(event.Path)
				}
			}
		case err := <-w.Errors:
			log.Println("Watch error: ", err)
		case <-quit:
			state := FsEvents.GetState()
			err := state.Save()
			return err // QUIT
		}
	}
}

func FileWatcher(quit chan bool) {
	cacheBpsThresholdEnv := os.Getenv("CACHE_BPS_THRESHOLD")
	if cacheBpsThresholdEnv != "" {
		num, err := strconv.ParseInt(cacheBpsThresholdEnv, 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		ThresholdCacheBps = float64(num) * 1000000.0
	}

	fnWatchPaths := os.Getenv("WATCH_PATHS")
	if fnWatchPaths == "" {
		fnWatchPaths = "watch.cnf"
	}

	fi, err := os.Open(fnWatchPaths)
	if err != nil {
		log.Fatal(err)
	}
	defer fi.Close()

	var mask uint32 = fsevents.AllEvents

	w, err := fsevents.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(fi)
	for scanner.Scan() {
		watchDir := scanner.Text()
		watchDir = strings.TrimSpace(watchDir)

		if watchDir != "" {
			d, err := w.AddDescriptor(watchDir, mask)
			if err != nil {
				log.Fatalf("%v (%v)", err, watchDir)
			}

			if err := d.Start(); err != nil {
				log.Fatalf("%v (%v)", err, watchDir)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	if err := handleEvents(w, quit); err != nil {
		log.Fatalf("Error handling events: %s", err.Error())
	}
}
