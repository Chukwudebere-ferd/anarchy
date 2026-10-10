package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"anarchy/internal/character"
	"anarchy/internal/evaluation"
	"anarchy/internal/execution"
	"anarchy/internal/identity"
	"anarchy/internal/integrations"
	"anarchy/internal/platform"
	"anarchy/internal/training"

	"github.com/jackc/pgx/v5/pgxpool"
)

func loadJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func loadDotEnv() {
	if os.Getenv("DATABASE_URL") != "" && os.Getenv("ANARCHY_CONTACT_KEY") != "" {
		return
	}
	b, err := os.ReadFile(".env")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		i := strings.Index(line, "=")
		if i < 0 {
			continue
		}
		k, v := strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:])
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
}

func main() {
	loadDotEnv()
	idsPath := flag.String("identities", "functions/identities.json", "path to identities JSON array")
	charsPath := flag.String("characters", "functions/characters.json", "path to characters JSON array")
	count := flag.Int("n", 10, "number of randomised bots to generate")
	out := flag.String("out", "functions/export.json", "output path for exportable bots")
	seed := flag.Int64("seed", time.Now().UnixNano(), "random seed")
	useDB := flag.Bool("use-db", false, "load identities/characters from Postgres (DATABASE_URL)")
	importToDB := flag.Bool("import-to-db", false, "import JSON files into Postgres before generating")
	migrate := flag.Bool("migrate", false, "apply migrations then exit")
	schedule := flag.String("schedule", "@every 2h", "schedule window stamped on deployments")
	jitter := flag.Int("jitter", 600, "jitter seconds spreading scheduled runs")
	evalOnly := flag.Bool("eval", false, "score character training data and exit")
	demoHold := flag.Bool("demo-hold", false, "demonstrate the approval gate and exit")
	statsOnly := flag.Bool("stats", false, "print inventory counts from Postgres and exit")
	bundleDir := flag.String("bundle", "", "write OpenCode company bundle to this directory")
	company := flag.String("company", "pilot", "company name stamped into the bundle")
	importChars := flag.String("import-chars", "", "bulk-import characters JSON via fast path, then continue")
	reset := flag.Bool("reset", false, "truncate tasks+deployments before generating (identities/characters kept)")
	flag.Parse()

	if *demoHold {
		runDemoHold()
		return
	}

	if *statsOnly {
		ctx := context.Background()
		pool, err := platform.Connect(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "db connect:", err)
			os.Exit(1)
		}
		defer pool.Close()
		var nIds, nChars, nDeps, nTasks int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM identities WHERE disabled=false AND deleted_at IS NULL`).Scan(&nIds)
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM characters`).Scan(&nChars)
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM deployments`).Scan(&nDeps)
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM tasks`).Scan(&nTasks)
		fmt.Printf("inventory: identities=%d characters=%d deployments=%d tasks=%d\n", nIds, nChars, nDeps, nTasks)
		fmt.Printf("max combinations: %d identities x %d characters\n", nIds, nChars)
		return
	}

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

	if _, err := execution.ParseSchedule(*schedule); err != nil {
		fmt.Fprintln(os.Stderr, "schedule:", err)
		os.Exit(1)
	}

	var pool *pgxpool.Pool
	if *useDB || *importToDB || *migrate || *importChars != "" {
		ctx := context.Background()
		var err error
		pool, err = platform.Connect(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "db connect:", err)
			os.Exit(1)
		}
		defer pool.Close()
		if *migrate {
			for _, f := range []string{"migrations/0001_init.sql", "migrations/0002_age.sql", "migrations/0003_schedules.sql", "migrations/0004_npc.sql"} {
				sql, err := os.ReadFile(f)
				if err != nil {
					fmt.Fprintln(os.Stderr, "read migration "+f+":", err)
					os.Exit(1)
				}
				if _, err := pool.Exec(ctx, string(sql)); err != nil {
					fmt.Fprintln(os.Stderr, "apply migration "+f+":", err)
					os.Exit(1)
				}
			}
			fmt.Println("migrations applied")
			if !*useDB && !*importToDB {
				return
			}
		}
		key, err := platform.LoadContactKey()
		if err != nil {
			fmt.Fprintln(os.Stderr, "contact key:", err)
			os.Exit(1)
		}
		if *importToDB {
			for _, id := range ids {
				if err := identity.Save(ctx, pool, key, id); err != nil {
					fmt.Fprintln(os.Stderr, "save identity:", err)
					os.Exit(1)
				}
			}
			for _, ch := range chars {
				if err := character.Save(ctx, pool, ch); err != nil {
					fmt.Fprintln(os.Stderr, "save character:", err)
					os.Exit(1)
				}
			}
			fmt.Printf("imported %d identities, %d characters to Postgres\n", len(ids), len(chars))
		}
		if *importChars != "" {
			var bulkChars []character.Character
			if err := loadJSON(*importChars, &bulkChars); err != nil {
				fmt.Fprintln(os.Stderr, "load bulk characters:", err)
				os.Exit(1)
			}
			if err := character.SaveAll(ctx, pool, bulkChars); err != nil {
				fmt.Fprintln(os.Stderr, "save bulk characters:", err)
				os.Exit(1)
			}
			fmt.Printf("imported %d characters to Postgres (bulk)\n", len(bulkChars))
		}
		if *useDB {
			dbIDs, err := identity.ListActive(ctx, pool, key)
			if err != nil {
				fmt.Fprintln(os.Stderr, "list identities:", err)
				os.Exit(1)
			}
			dbChars, err := character.ListAll(ctx, pool)
			if err != nil {
				fmt.Fprintln(os.Stderr, "list characters:", err)
				os.Exit(1)
			}
			ids, chars = dbIDs, dbChars
			fmt.Printf("loaded %d identities, %d characters from Postgres\n", len(ids), len(chars))
		}
	}

	if *evalOnly {
		failed := 0
		total := 0
		for _, ch := range chars {
			for i, r := range evaluation.EvaluateCharacter(ch) {
				total++
				status := "PASS"
				if !r.Passed {
					status = "FAIL"
					failed++
				}
				fmt.Printf("[%s] %s v%d example %d (%s): score=%.2f %s\n",
					status, r.CharacterID, r.CharacterVersion, i+1, r.ScenarioID, r.Score, r.Notes)
			}
		}
		fmt.Printf("eval: %d/%d passed\n", total-failed, total)
		if failed > 0 {
			os.Exit(1)
		}
		return
	}

	rng := rand.New(rand.NewSource(*seed))
	deps, err := training.RandomPool(rng, ids, chars, *count)
	if err != nil {
		fmt.Fprintln(os.Stderr, "randomise:", err)
		os.Exit(1)
	}
	deps = training.AssignSchedule(deps, *schedule, *jitter)

	if pool != nil && *useDB {
		ctx := context.Background()
		if *reset {
			if err := training.ResetDeployments(ctx, pool); err != nil {
				fmt.Fprintln(os.Stderr, "reset deployments:", err)
				os.Exit(1)
			}
			fmt.Println("deployments reset")
		}
		if err := training.SaveDeployments(ctx, pool, deps); err != nil {
			fmt.Fprintln(os.Stderr, "save deployments:", err)
			os.Exit(1)
		}
		n, _ := training.CountDeployments(ctx, pool)
		fmt.Printf("stored %d deployments (%d total)\n", len(deps), n)
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
	fmt.Printf("generated %d bots -> %s (seed=%d schedule=%s jitter=%ds)\n", len(exports), *out, *seed, *schedule, *jitter)
	for _, e := range exports {
		fmt.Printf(" - %s : identity=%s x character=%s v%d\n", e.DeploymentID, e.IdentityName, e.Character.Name, e.Character.Version)
	}
	if *bundleDir != "" {
		if err := integrations.WriteBundle(*bundleDir, *company, deps); err != nil {
			fmt.Fprintln(os.Stderr, "write bundle:", err)
			os.Exit(1)
		}
		fmt.Printf("bundle written -> %s (company=%s)\n", *bundleDir, *company)
	}
}

func runDemoHold() {
	fmt.Println("demo: autonomous mock task")
	t1, _ := execution.New("task-demo-1", "dep-demo", "signup",
		`{"username":"ada","email":"ada@example.com"}`, "@every 2h")
	fmt.Printf("  type=%s needs_approval=%v runnable=%v check=%v\n",
		t1.Type, t1.NeedsApproval, t1.Runnable(), execution.CheckTask(t1))

	fmt.Println("demo: sensitive task without approval")
	t2, _ := execution.New("task-demo-2", "dep-demo", "financial",
		`{"amount": 500}`, "@every 2h")
	fmt.Printf("  type=%s needs_approval=%v runnable=%v check=%v\n",
		t2.Type, t2.NeedsApproval, t2.Runnable(), execution.CheckTask(t2))

	fmt.Println("demo: client approves the sensitive task")
	if err := t2.Approve("startup-owner"); err != nil {
		fmt.Fprintln(os.Stderr, "approve:", err)
		os.Exit(1)
	}
	fmt.Printf("  type=%s needs_approval=%v runnable=%v check=%v approved_by=%s\n",
		t2.Type, t2.NeedsApproval, t2.Runnable(), execution.CheckTask(t2), t2.ApprovedBy)
}
