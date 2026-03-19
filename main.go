package main

import (
	"flag"
	"fmt"
	"spidersearch/internal/crawler"
	"spidersearch/internal/index"
	"spidersearch/internal/ui"
)

func main() {
	webPort := flag.Int("port", 8080, "Port for the Web UI")
	flag.Parse()

	fmt.Println("--- SpiderSearch (Brightwave Edition) ---")
	fmt.Printf("Web UI available at http://localhost:%d\n", *webPort)

	fi := index.NewFileIndex()
	jm := crawler.NewJobManager(fi)
	
	// Load past jobs from persistence
	jm.LoadPreviousJobs(".")

	web := ui.NewWebUI(jm, fi, *webPort)

	// Start Web UI (this will block until Exit or Error)
	if err := web.Start(); err != nil {
		fmt.Printf("Web UI Error: %v\n", err)
	}
}