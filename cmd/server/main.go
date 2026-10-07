package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"anarchy/internal/character"
	"anarchy/internal/identity"
	"anarchy/internal/integrations"
	"anarchy/internal/training"
)

func loadJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func main() {
	idsPath := flag.String("identities", "examples/identities.json", "path to identities JSON array")
	charsPath := flag.String("characters", "examples/characters.json", "path to characters JSON array")
	count := flag.Int("n", 10, "number of randomised bots to generate")
	out := flag.String("out", "examples/export.json", "output path for exportable bots")
	seed := flag.Int64("seed", time.Now().UnixNano(), "random seed")
	flag.Parse()

	var ids []identity.BotIdentity
	var chars []character.Character
	if err := loadJSON(*idsPath, &ids); err != nil {
		fmt.Fprintln(os.Stderr, "load identities:", err)
		os.Exit(1)
	}
	if err := loadJSON(*charsPath, &chars); err != nil {
		fmt.Fprintln(os.Stderr, "load characters:", err)
		os.Exit(1)
	}

	rng := rand.New(rand.NewSource(*seed))
	deps, err := training.RandomPool(rng, ids, chars, *count)
	if err != nil {
		fmt.Fprintln(os.Stderr, "randomise:", err)
		os.Exit(1)
	}

	exports := make([]integrations.ExportableBot, 0, len(deps))
	for _, d := range deps {
		exports = append(exports, integrations.ToExport(d, integrations.DefaultPermissions()))
	}

	b, _ := json.MarshalIndent(exports, "", "  ")
	if err := os.WriteFile(*out, b, 0644); err != nil {
		fmt.Fprintln(os.Stderr, "write out:", err)
		os.Exit(1)
	}
	fmt.Printf("generated %d bots -> %s (seed=%d)\n", len(exports), *out, *seed)
	for _, e := range exports {
		fmt.Printf(" - %s : identity=%s x character=%s v%d\n", e.DeploymentID, e.IdentityName, e.Character.Name, e.Character.Version)
	}
}
