package main

import (
	"github.com/infrasonar/go-libagent"
)

func CheckFs(_ *libagent.Check) (map[string][]map[string]any, error) {
	state := map[string][]map[string]any{}

	latest := FsEvents.GetState(200)

	FsEvents.Save()

	state["fileevents"] = latest

	state["agent"] = []map[string]any{{
		"name":    "fsevent",
		"version": version,
	}}

	return state, nil
}
