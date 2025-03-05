package main

import (
	"log"

	"github.com/infrasonar/go-libagent"
)

func main() {
	// Start collector
	log.Printf("Starting InfraSonar FileSystem Agent Collector v%s\n", version)

	// Initialize random
	libagent.RandInit()

	// Initialize Helper
	// libagent.GetHelper()

	// Set-up signal handler
	quit := make(chan bool)
	go libagent.SigHandler(quit)

	// Create Collector
	// collector := libagent.NewCollector("fsevent", version)

	// Create Asset
	// asset := libagent.NewAsset(collector)

	// asset.Kind = "Linux"
	// asset.Announce()

	// Create and plan checks
	// checkFs := libagent.Check{
	// 	Key:             "fs",
	// 	Collector:       collector,
	// 	Asset:           asset,
	// 	IntervalEnv:     "CHECK_FS",
	// 	DefaultInterval: 300,
	// 	NoCount:         false,
	// 	SetTimestamp:    false,
	// 	Fn:              CheckFs,
	// }
	// go checkFs.Plan(quit)

	FileWatcher(quit)
}
