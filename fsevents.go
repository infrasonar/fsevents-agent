package main

import (
	"bufio"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	fsevents "github.com/tywkeene/go-fsevents"
)

// THRESHOLD_NON_CACHE is the minimal time in seconds which we consider that the file
// is loaded from tape and not from cache
const THRESHOLD_NON_CACHE = 5.0

type FileEvent struct {
	Path            string
	LastTime        time.Time
	LastDuration    time.Duration
	LongestTime     time.Time
	LongestDuration time.Duration
}

type FileEventOut struct {
	Path            string  `json:"name"`
	LastTime        int64   `json:"lastTime"`
	LastDuration    float64 `json:"lastDuration"`
	LongestTime     int64   `json:"longestTime"`
	LongestDuration float64 `json:"longestDuration"`
}

type FsEventsStore struct {
	register map[string]*FileEvent
	n        int64
	average  float64
}

var FsEvents = FsEventsStore{
	register: map[string]*FileEvent{},
	n:        0,
}

func (f *FsEventsStore) Set(path string) {
	if v, ok := f.register[path]; ok {
		v.LastTime = time.Now()
		v.LastDuration = 0 // Must reset to 0
	} else {
		f.register[path] = &FileEvent{
			Path:     path,
			LastTime: time.Now(),
		}
	}
}

func (f *FsEventsStore) Upd(path string) {
	if v, ok := f.register[path]; ok {
		v.LastDuration = time.Since(v.LastTime)
		if v.LongestDuration < v.LastDuration {
			v.LongestDuration = v.LastDuration
			v.LongestTime = v.LastTime
		}
		elapsed := v.LongestDuration.Seconds()
		if elapsed > THRESHOLD_NON_CACHE {
			nn := float64(f.n)
			f.n += 1
			f.average = (f.average*nn + elapsed) / float64(f.n)
		}
	}
}

// Latest returns the latest `n“ files which are processed (queued files are skipped)
// This also cleans older files from the registry so we don't need to clean during each
// registration
func (f *FsEventsStore) Latest(n int) []*FileEventOut {
	done := make([]*FileEvent, 0, len(f.register))
	ret := make([]*FileEventOut, 0, n)

	for _, v := range f.register {
		if !v.LongestTime.IsZero() {
			done = append(done, v)
		}
	}

	slices.SortFunc(done, func(a, b *FileEvent) int {
		if a.LongestTime.Before(b.LongestTime) {
			return 1
		}
		if a.LongestTime.After(b.LongestTime) {
			return -1
		}
		return 0
	})

	for idx, v := range done {
		if idx > n {
			delete(f.register, v.Path)
		} else {
			ret = append(ret, &FileEventOut{
				Path:            v.Path,
				LastTime:        v.LastTime.Unix(),
				LastDuration:    v.LastDuration.Seconds(),
				LongestTime:     v.LongestTime.Unix(),
				LongestDuration: v.LastDuration.Seconds(),
			})
		}
	}
	return ret
}

func handleEvents(w *fsevents.Watcher, quit chan bool) error {
	// Watch for events
	go w.Watch()
	for {
		select {
		case event := <-w.Events:
			if !event.IsDirEvent() {
				log.Printf("Path: \"%s\"    Mask: %d", event.Path, event.RawEvent.Mask)
				if fsevents.CheckMask(fsevents.Open, event.RawEvent.Mask) {
					FsEvents.Set(event.Path)
				} else {
					FsEvents.Upd(event.Path)
				}
			}
		case err := <-w.Errors:
			log.Println("Watch error: ", err)
		case <-quit:
			return nil // QUIT
		}
	}
}

func FileWatcher(quit chan bool) {
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
				log.Fatal(err)
			}

			if err := d.Start(); err != nil {
				log.Fatal(err)
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
