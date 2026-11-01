// Command export writes complete JSONL graphs, one per fixed profile.
package main

import (
	"encoding/json"
	"github.com/JakeFAU/alan/research/dispatchauthority"
	"log"
	"os"
)

func main() {
	out := json.NewEncoder(os.Stdout)
	for _, p := range dispatchauthority.Profiles() {
		if err := out.Encode(dispatchauthority.Explore(p)); err != nil {
			log.Fatal(err)
		}
	}
}
