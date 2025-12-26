package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	interactive := flag.Bool("i", false, "Start interactive SQL session")
	daemon := flag.Bool("d", false, "Start Mist as a MySQL-compatible daemon")
	daemonLong := flag.Bool("daemon", false, "Start Mist as a MySQL-compatible daemon")
	port := flag.Int("port", 3306, "Port for daemon mode")
	persist := flag.String("persist", "", "Path to AOF file for persistence (enabled if not empty)")
	snapshot := flag.String("snapshot", "", "Path to snapshot file")
	syncInterval := flag.String("sync", "always", "Sync interval for AOF: always, everysec, none")
	version := flag.Bool("version", false, "Show version information")

	flag.Parse()

	if *version {
		fmt.Printf("Mist version %s\n", mist.Version())
		return
	}

	options := mist.PersistenceOptions{
		Enabled:      *persist != "" || *snapshot != "",
		AofPath:      *persist,
		SnapshotPath: *snapshot,
		SyncInterval: *syncInterval,
	}

	if *daemon || *daemonLong {
		fmt.Printf("Starting Mist daemon on port %d...\n", *port)
		if options.Enabled {
			fmt.Printf("Persistence enabled: %s (sync: %s)\n", options.AofPath, options.SyncInterval)
		}
		err := mist.RunSimpleDaemonWithOptions(*port, options)
		if err != nil {
			log.Fatalf("Daemon failed: %v", err)
		}
		return
	}

	// Default to interactive or just print help
	engine := mist.NewSQLEngineWithOptions(options)
	if *interactive || (flag.NFlag() == 0) || (flag.NFlag() == 1 && options.Enabled) {
		mist.Interactive(engine)
	} else {
		flag.Usage()
	}
}
