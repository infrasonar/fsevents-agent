package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"
	"time"

	"github.com/infrasonar/go-libagent"
)

// STATE_VERSION is the version for tje json file. This might be useful when a
// migration is required.
const STATE_VERSION = 1

// THRESHOLD_NON_CACHE is the minimal time in seconds which we consider that the file
// is loaded from tape and not from cache. Longer than 8.0 seconds seems reasonable to
// assume that the file has been read from tape.
const THRESHOLD_NON_CACHE = 8.0

// ThresholdCacheBps can be overwritten with environment variable
var ThresholdCacheBps float64 = 800000000.0

// MAX_FILES is the maximum number of files considered as latest. The oldestfiles will be
// removed from cache once this limit is reached.
const MAX_FILES = 200

func isFromCache(elapsed, bps float64) bool {
	return elapsed < THRESHOLD_NON_CACHE || bps > ThresholdCacheBps
}

type FileEvent struct {
	Path               string
	LastTime           time.Time
	LastDuration       time.Duration
	LastFileSize       int64
	LastBytesPerSec    float64
	LongestTime        time.Time
	LongestDuration    time.Duration
	LongestFileSize    int64
	LongestBytesPerSec float64
}

type StateLoad struct {
	Version     int     `json:"version"`
	Average     float64 `json:"average"`
	AverageTape float64 `json:"averageTape"`
	Counter     int64   `json:"n"`
	CounterTape int64   `json:"nTape"`
	Latest      []struct {
		Path               string  `json:"name"`
		LastTime           int64   `json:"lastTime"`
		LastDuration       float64 `json:"lastDuration"`
		LastFileSize       int64   `json:"lastFileSize"`
		LastBytesPerSec    float64 `json:"lastBytesPerSec"`
		LongestTime        int64   `json:"longestTime"`
		LongestDuration    float64 `json:"longestDuration"`
		LongestFileSize    int64   `json:"longestFileSize"`
		LongestBytesPerSec float64 `json:"longestBytesPerSec"`
	} `json:"latest"`
}

type State struct {
	Version               int              `json:"version"`
	Average               float64          `json:"average"`
	AverageTape           float64          `json:"averageTape"`
	Counter               int64            `json:"n"`
	CounterTape           int64            `json:"nTape"`
	BytesTape             float64          `json:"bytesTape"`
	DurationTape          float64          `json:"durationTape"`
	Latest                []map[string]any `json:"latest"`
	avgLatest             float64
	avgLatestTape         float64
	bytesPerSecTape       float64
	bytesPerSecLatestTape float64
	numTape               int
}

type FsEventsStore struct {
	register     map[string]*FileEvent
	n            int64
	nTape        int64
	average      float64
	averageTape  float64
	bytesTape    float64
	durationTape float64
}

var FsEvents = FsEventsStore{
	register:     map[string]*FileEvent{},
	n:            0,
	nTape:        0,
	average:      0.0,
	averageTape:  0.0,
	bytesTape:    0.0,
	durationTape: 0.0,
}

func (f *FsEventsStore) Set(path string) {
	fmt.Printf("Register path: %s\n", path)
	if v, ok := f.register[path]; ok {
		// Reset to time and zero to all
		v.LastTime = time.Now()
		v.LastDuration = 0
		v.LastFileSize = 0
		v.LastBytesPerSec = 0.0
	} else {
		f.register[path] = &FileEvent{
			Path:     path,
			LastTime: time.Now(),
		}
	}
}

func (f *FsEventsStore) Upd(path string) {
	if v, ok := f.register[path]; ok {
		fi, err := os.Stat(path)
		if err == nil {
			fmt.Printf("Calculate time path: %s\n", path)
			v.LastFileSize = fi.Size()
			v.LastDuration = time.Since(v.LastTime)

			elapsed := v.LastDuration.Seconds()
			v.LastBytesPerSec = float64(v.LastFileSize) / elapsed

			if !isFromCache(elapsed, v.LastBytesPerSec) {
				nn := float64(f.nTape)
				f.nTape += 1
				f.averageTape = (f.averageTape*nn + elapsed) / float64(f.nTape)
				f.bytesTape += float64(v.LastFileSize)
				f.durationTape += elapsed
			}

			if v.LongestDuration < v.LastDuration {
				v.LongestDuration = v.LastDuration
				v.LongestTime = v.LastTime
				v.LongestFileSize = v.LastFileSize
				v.LongestBytesPerSec = v.LastBytesPerSec
			}

			nn := float64(f.n)
			f.n += 1
			f.average = (f.average*nn + elapsed) / float64(f.n)
		} else {
			log.Printf("Failed to read file stat: %v (%v)", path, err)
		}
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
	bytesPerSecTape := 0.0
	bytesPerSecLatestTape := 0.0

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
			lastFromCache := isFromCache(lastDuration, v.LastBytesPerSec)
			longestDuration := v.LongestDuration.Seconds()
			longestFromCache := isFromCache(longestDuration, v.LongestBytesPerSec)

			ret = append(ret, map[string]any{
				"name":               v.Path,
				"lastTime":           v.LastTime.Unix(),
				"lastDuration":       libagent.IFloat64(lastDuration),
				"lastBytesPerSec":    libagent.IFloat64(v.LastBytesPerSec),
				"lastFileSize":       v.LastFileSize,
				"lastFromCache":      lastFromCache,
				"longestTime":        v.LongestTime.Unix(),
				"longestDuration":    libagent.IFloat64(longestDuration),
				"longestBytesPerSec": libagent.IFloat64(v.LongestBytesPerSec),
				"longestFileSize":    v.LongestFileSize,
				"longestFromCache":   longestFromCache,
			})
			avgLatest += longestDuration
			if !longestFromCache {
				numTape += 1
				avgLatestTape += longestDuration
				bytesPerSecLatestTape += float64(v.LongestFileSize)
			}
		}
	}

	if avgLatest > 0.0 {
		avgLatest /= float64(len(ret))
	}
	if avgLatestTape > 0.0 {
		bytesPerSecLatestTape /= avgLatestTape
		avgLatestTape /= float64(numTape)
	}
	if f.durationTape > 0.0 {
		bytesPerSecTape = f.bytesTape / f.durationTape
	}

	return &State{
		Version:               STATE_VERSION,
		Average:               f.average,
		AverageTape:           f.averageTape,
		Counter:               f.n,
		CounterTape:           f.nTape,
		BytesTape:             f.bytesTape,
		DurationTape:          f.durationTape,
		Latest:                ret,
		avgLatest:             avgLatest,
		avgLatestTape:         avgLatestTape,
		bytesPerSecTape:       bytesPerSecTape,
		bytesPerSecLatestTape: bytesPerSecLatestTape,
		numTape:               numTape,
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
