package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/topbase/topbase/internal/backup"
	"github.com/topbase/topbase/internal/buildinfo"
)

func main() {
	dataDir := os.Getenv("TOPBASE_DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	now := time.Now().UTC()
	destination := ""
	if len(os.Args) > 2 {
		log.Fatalf("usage: %s [destination-directory]", filepath.Base(os.Args[0]))
	}
	if len(os.Args) == 2 {
		destination = os.Args[1]
	} else {
		destination = filepath.Join("backups", "topbase-"+now.Format("20060102T150405Z"))
	}
	manifest, err := backup.Create(dataDir, destination, buildinfo.Version, now)
	if err != nil {
		log.Fatal(err)
	}
	abs, _ := filepath.Abs(destination)
	fmt.Printf("Topbase backup created: %s (%d files, version %s)\n", abs, len(manifest.Files), manifest.Version)
}
