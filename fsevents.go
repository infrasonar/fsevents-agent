package main

import (
	"bufio"
	"encoding/json"
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

// MAX_FILES is the maximum number of files considered as latest. The oldestfiles will be
// removed from cache once this limit is reached.
const MAX_FILES = 200

type FileEvent struct {
	Path            string
	LastTime        time.Time
	LastDuration    time.Duration
	LongestTime     time.Time
	LongestDuration time.Duration
}

type FileEventState struct {
	Path            string  `json:"name"`
	LastTime        int64   `json:"lastTime"`
	LastDuration    float64 `json:"lastDuration"`
	LongestTime     int64   `json:"longestTime"`
	LongestDuration float64 `json:"longestDuration"`
}
type StateLoad struct {
	Version int     `json:"version"`
	Average float64 `json:"average"`
	Counter int64   `json:"n"`
	Latest  []struct {
		Path            string  `json:"name"`
		LastTime        int64   `json:"lastTime"`
		LastDuration    float64 `json:"lastDuration"`
		LongestTime     int64   `json:"longestTime"`
		LongestDuration float64 `json:"longestDuration"`
	} `json:"latest"`
}
type State struct {
	Version int              `json:"version"`
	Average float64          `json:"average"`
	Counter int64            `json:"n"`
	Latest  []map[string]any `json:"latest"`
}

type FsEventsStore struct {
	register map[string]*FileEvent
	n        int64
	average  float64
}

var FsEvents = FsEventsStore{
	register: map[string]*FileEvent{},
	n:        0,
	average:  0.0,
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
func (f *FsEventsStore) GetState() *State {
	done := make([]*FileEvent, 0, len(f.register))
	ret := make([]map[string]any, 0, MAX_FILES)

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
		if idx > MAX_FILES {
			delete(f.register, v.Path)
		} else {
			ret = append(ret, map[string]any{
				"name":            v.Path,
				"lastTime":        v.LastTime.Unix(),
				"lastDuration":    v.LastDuration.Seconds(),
				"longestTime":     v.LongestTime.Unix(),
				"longestDuration": v.LastDuration.Seconds(),
			})
		}
	}
	return &State{
		Version: 0,
		Average: f.average,
		Counter: f.n,
		Latest:  ret,
	}
}
func (s *State) Save() error {
	fn := os.Getenv("FN_STATE_JSON")
	if fn == "" {
		fn = "state.json"
	}

	bytes, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fn, bytes, 0644)
}

func (f *FsEventsStore) Restore() {
	fn := os.Getenv("FN_STATE_JSON")
	if fn == "" {
		fn = "state.json"
	}

	content, err := os.ReadFile(fn)
	if err != nil {
		log.Printf("Error reading state file: %v", err)
		return
	}

	var payload StateLoad
	err = json.Unmarshal(content, &payload)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
	}
	f.average = payload.Average
	f.n = payload.Counter
	for _, v := range payload.Latest {
		f.register[v.Path] = &FileEvent{
			Path:            v.Path,
			LastTime:        time.Unix(v.LastTime, 0),
			LastDuration:    time.Duration(v.LastDuration * float64(time.Second)),
			LongestTime:     time.Unix(v.LongestTime, 0),
			LongestDuration: time.Duration(v.LongestDuration * float64(time.Second)),
		}
	}
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
