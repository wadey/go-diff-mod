package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
)

var (
	flagDirect   = flag.Bool("direct", false, "Only list updates to direct dependencies.")
	flagIndirect = flag.Bool("indirect", false, "Only list updates to indirect dependencies.")
	flagMarkdown = flag.Bool("markdown", false, "Output in markdown format.")

	postfixVersion = regexp.MustCompile(`/(v[0-9]+)$`)
	pseudoVersion  = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+-(?:pre.0.|0.)?[0-9]{14}-([0-9a-f]{12}$)`)
)

type Module struct {
	Indirect bool
	Update   *Module
	Path     string
	Version  string

	httpPath string
}

func main() {
	flag.Parse()

	if flag.NArg() != 2 {
		log.Fatalf(`usage: go-diff-mod <old json> <new json>

Directions:

  1. Run "go list -m -json all | jq -s . >before.json"
  2. Do your package upgrades / installs
  3. Run "go list -m -json all | jq -s . >after.json"
  4. Run "go-diff-mod before.json after.json"
`)
	}

	oldRaw, err := ioutil.ReadFile(flag.Arg(0))
	if err != nil {
		log.Fatalf("failed to read %s: %v", flag.Arg(0), err)
	}
	newRaw, err := ioutil.ReadFile(flag.Arg(1))
	if err != nil {
		log.Fatalf("failed to read %s: %v", flag.Arg(1), err)
	}

	var oldModules, newModules []*Module
	if err = json.Unmarshal(oldRaw, &oldModules); err != nil {
		log.Fatalf("failed to parse %s: %v", flag.Arg(0), err)
	}
	if err = json.Unmarshal(newRaw, &newModules); err != nil {
		log.Fatalf("failed to parse %s: %v", flag.Arg(1), err)
	}

	oldMap := map[string]*Module{}
	for _, m := range oldModules {
		if postfixVersion.MatchString(m.Path) {
			// TODO proper to always strip?
			m.Path = m.Path[0:strings.LastIndex(m.Path, "/")]
		}
		oldMap[m.Path] = m
	}
	newMap := map[string]*Module{}
	for _, m := range newModules {
		if postfixVersion.MatchString(m.Path) {
			// TODO proper to always strip?
			m.Path = m.Path[0:strings.LastIndex(m.Path, "/")]
		}
		newMap[m.Path] = m
	}

	var changed, added, removed []*Module

	for _, oldM := range oldModules {
		newM := newMap[oldM.Path]
		if newM != nil {
			if newM.Version != oldM.Version {
				oldM.Update = newM
				changed = append(changed, oldM)
			}
		} else {
			removed = append(removed, oldM)
		}
	}
	for _, newM := range newModules {
		oldM := oldMap[newM.Path]
		if oldM == nil {
			added = append(added, newM)
		}
	}

	for _, m := range changed {
		output("Updated", m)
	}
	for _, m := range added {
		output("Added", m)
	}
	for _, m := range removed {
		output("Removed", m)
	}
}

func output(label string, m *Module) {
	if !(*flagDirect && m.Indirect) && !(*flagIndirect && !m.Indirect) {
		m.Version = pseudoVersion.ReplaceAllString(m.Version, "$1")

		m.Version = strings.TrimSuffix(m.Version, "+incompatible")

		switch {
		case strings.HasPrefix(m.Path, "golang.org/x/"):
			m.httpPath = "github.com/golang/" + strings.TrimPrefix(m.Path, "golang.org/x/")
		case strings.HasPrefix(m.Path, "k8s.io/"):
			m.httpPath = "github.com/kubernetes/" + strings.TrimPrefix(m.Path, "k8s.io/")
		case m.Path == "google.golang.org/protobuf":
			m.httpPath = "github.com/protocolbuffers/protobuf-go"
		default:
			m.httpPath = m.Path
		}

		if m.Update != nil {
			m.Update.Version = pseudoVersion.ReplaceAllString(m.Update.Version, "$1")

			switch {
			case strings.HasPrefix(m.httpPath, "github.com/"):
				if *flagMarkdown {
					fmt.Printf("- %s [%s (%s -> %s)](https://%s/compare/%s...%s)\n", label, m.Path, m.Version, m.Update.Version, m.httpPath, m.Version, m.Update.Version)
				} else {
					fmt.Printf("%s\t%s\thttps://%s/compare/%s...%s\n", label, m.Path, m.httpPath, m.Version, m.Update.Version)
				}
			default:
				if *flagMarkdown {
					fmt.Printf("- %s %s (%s -> %s)\n", label, m.Path, m.Version, m.Update.Version)
				} else {
					fmt.Printf("%s\t%s\t%s...%s\n", label, m.Path, m.Version, m.Update.Version)
				}
			}
		} else {
			switch {
			case strings.HasPrefix(m.httpPath, "github.com/"):
				if *flagMarkdown {
					fmt.Printf("- %s [%s (%s)](https://%s/tree/%s)\n", label, m.Path, m.Version, m.httpPath, m.Version)
				} else {
					fmt.Printf("%s\t%s\thttps://%s/tree/%s\n", label, m.Path, m.httpPath, m.Version)
				}
			default:
				if *flagMarkdown {
					fmt.Printf("- %s %s (%s)\n", label, m.Path, m.Version)
				} else {
					fmt.Printf("%s\t%s\t%s\n", label, m.Path, m.Version)
				}
			}
		}
	}
}
