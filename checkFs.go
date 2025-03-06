package main

import (
	"github.com/infrasonar/go-libagent"
)

func 

func CheckFs(_ *libagent.Check) (map[string][]map[string]any, error) {
	state := map[string][]map[string]any{}

	state["agent"] = []map[string]any{{
		"name":    "fsevent",
		"version": version,
	}}

	return state, nil
}
