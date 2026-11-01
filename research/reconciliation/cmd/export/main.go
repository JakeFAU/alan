package main

import (
	"encoding/json"
	"os"

	"github.com/JakeFAU/alan/research/reconciliation"
)

func main() {
	enc := json.NewEncoder(os.Stdout)
	for _, p := range reconciliation.Profiles() {
		if err := enc.Encode(reconciliation.Explore(p)); err != nil {
			panic(err)
		}
	}
}
