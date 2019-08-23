package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var (
	flagDirect = flag.Bool("direct", false, "Only list updates to direct dependencies.")

	pseudoVersion = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+-(?:pre.0.|0.)?[0-9]{14}-([0-9a-f]{12}$)`)
)

type Module struct {
	Indirect bool
	Update   *Module
	Path     string
	Version  string
}

func main() {
	flag.Parse()

	cmd := exec.Command("go", "list", "-u", "-m", "-json", "all")
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("failed to run `go list`: %v", err)
	}

	if err = cmd.Start(); err != nil {
		log.Fatalf("failed to run `go list`: %v", err)
	}

	dec := json.NewDecoder(stdout)

	for {
		var m Module
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("failed to decode json: %v", err)
		}

		if !(*flagDirect && m.Indirect) && m.Update != nil {
			m.Version = pseudoVersion.ReplaceAllString(m.Version, "$1")
			m.Update.Version = pseudoVersion.ReplaceAllString(m.Update.Version, "$1")

			if strings.HasPrefix(m.Path, "golang.org/x/") {
				m.Path = "github.com/golang/" + strings.TrimPrefix(m.Path, "golang.org/x/")
			}

			switch {
			case strings.HasPrefix(m.Path, "github.com/"):
				fmt.Printf("https://%s/compare/%s...%s\n", m.Path, m.Version, m.Update.Version)
			default:
				fmt.Printf("%s - %s - %s\n", m.Path, m.Version, m.Update.Version)
			}
		}
	}

	if err = cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}
