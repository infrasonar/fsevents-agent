package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"time"
)

// THRESHOLD_NON_CACHE is the minimal time in seconds which we consider that the file
// is loaded from tape and not from cache. Longer than 8.0 seconds seems reasonable to
// assume that the file has been read from tape.
const THRESHOLD_NON_CACHE = 8.0

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
	Version     int     `json:"version"`
	Average     float64 `json:"average"`
	AverageTape float64 `json:"averageTape"`
	Counter     int64   `json:"n"`
	CounterTape int64   `json:"nTape"`
	Latest      []struct {
		Path            string  `json:"name"`
		LastTime        int64   `json:"lastTime"`
		LastDuration    float64 `json:"lastDuration"`
		LongestTime     int64   `json:"longestTime"`
		LongestDuration float64 `json:"longestDuration"`
	} `json:"latest"`
}

type State struct {
	Version       int              `json:"version"`
	Average       float64          `json:"average"`
	AverageTape   float64          `json:"averageTape"`
	Counter       int64            `json:"n"`
	CounterTape   int64            `json:"nTape"`
	Latest        []map[string]any `json:"latest"`
	avgLatest     float64
	avgLatestTape float64
	numTape       int
}

type FsEventsStore struct {
	register    map[string]*FileEvent
	n           int64
	nTape       int64
	average     float64
	averageTape float64
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
		if elapsed >= THRESHOLD_NON_CACHE {
			nn := float64(f.nTape)
			f.nTape += 1
			f.averageTape = (f.averageTape*nn + elapsed) / float64(f.nTape)
		}
		nn := float64(f.n)
		f.n += 1
		f.average = (f.average*nn + elapsed) / float64(f.n)
	}
}

// Latest returns the latest `n“ files which are processed (queued files are skipped)
// This also cleans older files from the registry so we don't need to clean during each
// registration
func (f *FsEventsStore) GetState() *State {
	done := make([]*FileEvent, 0, len(f.register))
	ret := make([]map[string]any, 0, MAX_FILES)
	avgLatest := 0.0
	avgLatestTape := 0.0
	numTape := 0

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
			lastDuration := v.LastDuration.Seconds()
			lastFromCache := lastDuration < THRESHOLD_NON_CACHE
			longestDuration := v.LongestDuration.Seconds()
			longestFromCache := longestDuration < THRESHOLD_NON_CACHE

			ret = append(ret, map[string]any{
				"name":             v.Path,
				"lastTime":         v.LastTime.Unix(),
				"lastDuration":     lastDuration,
				"lastFromCache":    lastFromCache,
				"longestTime":      v.LongestTime.Unix(),
				"longestDuration":  longestDuration,
				"longestFromCache": longestFromCache,
			})
			avgLatest += longestDuration
			if !longestFromCache {
				numTape += 1
				avgLatestTape += longestDuration
			}
		}
	}

	if avgLatest > 0.0 {
		avgLatest /= float64(len(ret))
	}
	if avgLatestTape > 0.0 {
		avgLatestTape /= float64(numTape)
	}

	return &State{
		Version:       0,
		Average:       f.average,
		Counter:       f.n,
		Latest:        ret,
		avgLatest:     avgLatest,
		avgLatestTape: avgLatestTape,
		numTape:       numTape,
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

func (f *FsEventsStore) Restore() error {
	fn := os.Getenv("FN_STATE_JSON")
	if fn == "" {
		fn = "state.json"
	}

	content, err := os.ReadFile(fn)
	if err != nil {
		return fmt.Errorf("error reading state file: %v", err)
	}

	var payload StateLoad
	err = json.Unmarshal(content, &payload)
	if err != nil {
		return fmt.Errorf("error during Unmarshal(): %v", err)
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

	return nil
}
