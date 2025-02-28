package main

import (
	"log"
	"os"

	"github.com/infrasonar/go-libagent"
	fsevents "github.com/tywkeene/go-fsevents"
)

func handleEvents(w *fsevents.Watcher, quit chan bool) error {

	// Watch for events
	go w.Watch()
	log.Println("Waiting for events...")

	for {
		select {
		case event := <-w.Events:
			// Contextual metadata is stored in the event object as well as a pointer to the WatchDescriptor that event belongs to
			log.Printf("Event Name: %s Event Path: %s Event Descriptor: %v", event.Name, event.Path, event.Descriptor)
			// A Watcher keeps a running atomic counter of all events it sees
			log.Println("Watcher Event Count:", w.GetEventCount())
			log.Println("Running descriptors:", w.GetRunningDescriptors())
		case err := <-w.Errors:
			log.Println(err)
		case <-quit:
			log.Println("QUIT!!!")
			return nil
		}
	}
}

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

	if len(os.Args) == 1 {
		panic("Must specify directory to watch")
	}

	watchDir := os.Args[1]
	var mask uint32 = fsevents.AllEvents

	w, err := fsevents.NewWatcher()
	if err != nil {
		panic(err)
	}

	d, err := w.AddDescriptor(watchDir, mask)
	if err != nil {
		panic(err)
	}

	if err := d.Start(); err != nil {
		panic(err)
	}

	if err := handleEvents(w, quit); err != nil {
		log.Fatalf("Error handling events: %s", err.Error())
	}

	// Wait for quit
	//<-quit
}
