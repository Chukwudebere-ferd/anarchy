// Command seed generates bulk identity JSON for Anarchy.
//
// Sources:
//   local      (default) deterministic offline generator, reproducible via -seed.
//   randomuser fetches names/ages from https://randomuser.me (free, no key),
//              but contacts are ALWAYS rewritten to safe fakes (@example.com,
//              +1-555-... fictional range) so no real person can be contacted.
//
// Usage:
//
//	go run ./cmd/seed -n 200 -out examples/identities.bulk.json
//	go run ./cmd/seed -n 200 -source randomuser -out examples/identities.bulk.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"anarchy/internal/identity"
)

var firstNames = []string{
	"Ada", "Alex", "Amara", "Ben", "Chloe", "Darius", "Elena", "Ferd", "Grace",
	"Hassan", "Ingrid", "Jamal", "Kira", "Liam", "Maya", "Nadia", "Omar", "Priya",
	"Quinn", "Rosa", "Sophia", "Tunde", "Uma", "Victor", "Wren", "Yusuf", "Zara",
	"Diego", "Fatima", "George", "Hana", "Ivan", "Julia", "Kenji", "Lena",
	"Marco", "Nina", "Owen", "Paula", "Ravi",
}

var lastNames = []string{
	"Anderson", "Baker", "Chen", "Diallo", "Ellis", "Farah", "Garcia", "Haddad",
	"Ivers", "Johnson", "Kim", "Lopez", "Mensah", "Nguyen", "Okafor", "Patel",
	"Quintero", "Rossi", "Silva", "Tanaka", "Umar", "Vega", "Wang", "Xu",
	"Yamada", "Zhang", "Ali", "Brown", "Costa", "Dube", "Evans", "Fischer",
}

var roles = []string{
	"general saas operator", "onboarding specialist", "support triage agent",
	"back-office operator", "qa flow tester", "data entry operator",
}

func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "bot"
	}
	return b.String()
}

func makeEmail(rng *rand.Rand, first, last, domain string) string {
	f, l := slug(first), slug(last)
	num := rng.Intn(99) + 1
	switch rng.Intn(5) {
	case 0:
		return fmt.Sprintf("%s.%s%d@%s", f, l, num, domain)
	case 1:
		return fmt.Sprintf("%s_%s@%s", f, l, domain)
	case 2:
		return fmt.Sprintf("%s%s@%s", f[:1], l, domain)
	case 3:
		return fmt.Sprintf("%s%s%d@%s", f, l[:1], num, domain)
	default:
		return fmt.Sprintf("%s%s@%s", f, l, domain)
	}
}

func makePhone(rng *rand.Rand) string {
	switch rng.Intn(3) {
	case 0:
		return fmt.Sprintf("+1-555-%03d-%04d", rng.Intn(800)+200, rng.Intn(10000))
	case 1:
		return fmt.Sprintf("+1555%07d", rng.Intn(10000000))
	default:
		return fmt.Sprintf("(555) %03d-%04d", rng.Intn(800)+200, rng.Intn(10000))
	}
}

var descTemplates = []string{
	"%s, %d. %s with %d years across SaaS onboarding and support queues. Confirms before acting, escalates when unsure.",
	"%s (%d) — %s. %d years handling form-fill, triage and status lookups at volume while keeping tone consistent.",
	"Ops profile: %s, %d, %s. %d years in back-office SaaS flows; prefers short confirmations and clean handoffs.",
	"%s is %d and works as a %s with %d years behind them. Specialises in onboarding walkthroughs and FAQ-style Q&A.",
	"%s, %d — %s. %d years doing ticket triage and data entry; flags anything needing human approval instead of guessing.",
	"Meet %s (%d), %s with %d years in SaaS ops. Covers signup flows, settings resets and order-status checks; asks for clarification rather than assuming.",
}

func makeDescription(rng *rand.Rand, name string, age int, role string) string {
	years := age - 18
	if years < 1 {
		years = 1
	}
	if years > 30 {
		years = 20 + rng.Intn(11)
	}
	return fmt.Sprintf(descTemplates[rng.Intn(len(descTemplates))], name, age, role, years)
}

func makeIdentity(rng *rand.Rand, seq int, first, last string, age int, domain string) identity.BotIdentity {
	name := first + " " + last
	role := roles[rng.Intn(len(roles))]
	return identity.BotIdentity{
		ID:          fmt.Sprintf("id-gen-%05d", seq),
		Name:        name,
		Age:         age,
		Email:       makeEmail(rng, first, last, domain),
		Phone:       makePhone(rng),
		Profile:     role,
		Description: makeDescription(rng, name, age, role),
	}
}

func localBatch(rng *rand.Rand, start, n int, domain string) []identity.BotIdentity {
	out := make([]identity.BotIdentity, 0, n)
	for i := 0; i < n; i++ {
		first := firstNames[rng.Intn(len(firstNames))]
		last := lastNames[rng.Intn(len(lastNames))]
		age := rng.Intn(53) + 18 // 18-70
		out = append(out, makeIdentity(rng, start+i, first, last, age, domain))
	}
	return out
}

type randomUserResp struct {
	Results []struct {
		Name struct {
			First string `json:"first"`
			Last  string `json:"last"`
		} `json:"name"`
		DOB struct {
			Age int `json:"age"`
		} `json:"dob"`
	} `json:"results"`
}

func fetchRandomUserNames(n int) (pairs [][2]string, ages []int, err error) {
	client := &http.Client{Timeout: 15 * time.Second}
	for len(pairs) < n {
		want := n - len(pairs)
		if want > 100 {
			want = 100
		}
		url := fmt.Sprintf("https://randomuser.me/api/?results=%d&inc=name,dob&nat=us,gb,fr,de,br,in,ng,za&noinfo", want)
		resp, err := client.Get(url)
		if err != nil {
			return nil, nil, err
		}
		var r randomUserResp
		if derr := json.NewDecoder(resp.Body).Decode(&r); derr != nil {
			resp.Body.Close()
			return nil, nil, derr
		}
		resp.Body.Close()
		if len(r.Results) == 0 {
			return nil, nil, fmt.Errorf("randomuser.me returned no results")
		}
		for _, u := range r.Results {
			if u.Name.First == "" || u.Name.Last == "" {
				continue
			}
			age := u.DOB.Age
			if age < 18 || age > 90 {
				age = 18 + len(pairs)%53
			}
			pairs = append(pairs, [2]string{u.Name.First, u.Name.Last})
			ages = append(ages, age)
			if len(pairs) >= n {
				break
			}
		}
	}
	return pairs, ages, nil
}

func main() {
	n := flag.Int("n", 100, "number of identities to generate")
	out := flag.String("out", "functions/identities.bulk.json", "output JSON path")
	seed := flag.Int64("seed", time.Now().UnixNano(), "random seed (local source + emails/phones)")
	source := flag.String("source", "local", "source: local|randomuser")
	domain := flag.String("domain", "example.com", "email domain for generated contacts")
	start := flag.Int("start", 1, "starting id sequence number")
	flag.Parse()

	if *n < 1 {
		fmt.Fprintln(os.Stderr, "seed: -n must be >= 1")
		os.Exit(1)
	}
	rng := rand.New(rand.NewSource(*seed))

	var batch []identity.BotIdentity
	if *source == "randomuser" {
		pairs, ages, err := fetchRandomUserNames(*n)
		if err != nil {
			fmt.Fprintln(os.Stderr, "seed: randomuser.me fetch failed, falling back to local:", err)
			batch = localBatch(rng, *start, *n, *domain)
		} else {
			for i := range pairs {
				batch = append(batch, makeIdentity(rng, *start+i, pairs[i][0], pairs[i][1], ages[i], *domain))
			}
			fmt.Printf("seed: names/ages from randomuser.me, contacts on @%s\n", *domain)
		}
	} else {
		batch = localBatch(rng, *start, *n, *domain)
	}

	for i, b := range batch {
		if err := b.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "seed: generated identity %d invalid: %v\n", i, err)
			os.Exit(1)
		}
	}

	data, _ := json.MarshalIndent(batch, "", "  ")
	if err := os.WriteFile(*out, data, 0644); err != nil {
		fmt.Fprintln(os.Stderr, "seed: write:", err)
		os.Exit(1)
	}
	fmt.Printf("seed: wrote %d identities -> %s (source=%s seed=%d)\n", len(batch), *out, *source, *seed)
}
