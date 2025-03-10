package main

import (
	"github.com/infrasonar/go-libagent"
)

func CheckFs(_ *libagent.Check) (map[string][]map[string]any, error) {
	state := map[string][]map[string]any{}

	fstate := FsEvents.GetState()

	err := fstate.Save()

	// Latest are the N latest open file durations, where each item is like:
	// 		"name":             string 	(path)
	// 		"lastTime":         integer (unix timestamp)
	// 		"lastDuration":     float 	(duration in seconds)
	// 		"lastFromCache":    bool 	(loaded from cache versus from tape)
	// 		"longestTime":      integer	(unix timestamp)
	// 		"longestDuration":  float 	(duration in seconds)
	// 		"longestFromCache": bool 	(loaded from cache versus from tape)

	state["latest"] = fstate.Latest

	state["stats"] = []map[string]any{{
		"name":          "stats",
		"counter":       fstate.Counter,
		"counterTape":   fstate.CounterTape,
		"average":       fstate.Average,
		"averageTape":   fstate.AverageTape,
		"avgLatest":     fstate.avgLatest,
		"avgLatestTape": fstate.avgLatestTape,
		"numLatest":     len(fstate.Latest),
		"numLatestTape": fstate.numTape,
	}}

	state["agent"] = []map[string]any{{
		"name":    "fsevents",
		"version": version,
	}}

	// Note that err can be a "save" error; we want to know this as this would
	// be a problem when reloading the agent
	return state, err
}
