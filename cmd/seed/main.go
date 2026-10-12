// Command seed generates bulk identity JSON for Anarchy.
//
// Sources:
//
//	local      (default) deterministic offline generator, reproducible via -seed.
//	randomuser fetches names/ages from https://randomuser.me (free, no key),
//	           but contacts are ALWAYS rewritten to safe fakes (@example.com,
//	           +1-555-... fictional range) so no real person can be contacted.
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

	"anarchy/internal/character"
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
	attempts := 0
	for len(pairs) < n {
		attempts++
		if attempts > 10 {
			return nil, nil, fmt.Errorf("randomuser.me kept returning unusable results after %d tries", attempts-1)
		}
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
	fromIdentities := flag.String("from-identities", "", "character mode: generate 1:1 characters from this identities JSON instead of identities")
	charOut := flag.String("characters", "", "character mode output path (required with -from-identities)")
	flag.Parse()

	if *fromIdentities != "" {
		if *charOut == "" {
			fmt.Fprintln(os.Stderr, "seed: -characters output path is required with -from-identities")
			os.Exit(1)
		}
		generateCharacters(*fromIdentities, *charOut, *seed)
		return
	}

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

var traitPool = []string{
	"friendly", "patient", "proactive", "calm", "thorough", "polite",
	"precise", "efficient", "curious", "steady", "cheerful", "methodical",
	"adaptable", "diligent", "warm", "concise", "careful", "upbeat",
	"reliable", "observant",
}

var styleTemplates = []string{
	"Acts on schedule, logs outcomes. Content reads natural, never claims to be human.",
	"Steady and structured. Runs the schedule, records results, holds what needs approval.",
	"Short action logs. Runs high-volume schedules, flags holds clearly.",
	"Warm and consistent. Completes routine flows fully, pauses on anything sensitive.",
	"Quiet and reliable. Keeps activity evenly spread, reports anomalies plainly.",
	"Brisk, task-first worker. Minimal chatter, full outcome logs.",
}

var roleGoals = map[string][]string{
	"general saas operator": {"run scheduled platform tasks reliably", "keep mock data realistic and consistent", "hold sensitive actions for client approval"},
	"onboarding specialist": {"complete signup simulations accurately", "keep onboarding data consistent", "hold anything irreversible for approval"},
	"support triage agent":  {"sort simulated tickets consistently", "escalate uncertain cases", "never take destructive action unprompted"},
	"back-office operator":  {"process back-office batches on time", "log every outcome", "hold financial items for approval"},
	"qa flow tester":        {"exercise flows the way a new user would", "record failures precisely", "never alter production data"},
	"data entry operator":   {"enter data accurately at volume", "keep formatting consistent", "flag ambiguous records instead of guessing"},
}

var roleKnowledge = map[string][]string{
	"general saas operator": {"SaaS onboarding flows", "activity simulation", "order status flows"},
	"onboarding specialist": {"signup flows", "profile setup", "verification bypass handling"},
	"support triage agent":  {"ticket categories", "priority rules", "escalation paths"},
	"back-office operator":  {"batch processing", "record reconciliation", "status updates"},
	"qa flow tester":        {"user journeys", "edge cases", "bug reporting"},
	"data entry operator":   {"catalog seeding", "bulk data entry", "format standards"},
}

var holdPairs = [][2]string{
	{"task: refund order #4821", "held: financial action needs client approval first. Request prepared, not executed."},
	{"task: publish announcement sitewide", "held: external-publish needs client approval first. Draft staged, not sent."},
	{"task: delete inactive accounts", "held: destructive action needs client approval first. List prepared, nothing deleted."},
	{"task: message real users directly", "held: real-user-contact needs client approval first. Drafts prepared, nothing sent."},
}

var doneTasks = []string{
	"seed 20 product views", "run nightly signup simulation", "post scheduled update",
	"seed 50 catalog views with jitter", "simulate onboarding checklist", "run comment thread simulation",
}

func pickN(rng *rand.Rand, pool []string, n int) []string {
	idx := rng.Perm(len(pool))
	if n > len(idx) {
		n = len(idx)
	}
	out := make([]string, 0, n)
	for _, i := range idx[:n] {
		out = append(out, pool[i])
	}
	return out
}

func makeCharacter(rng *rand.Rand, seq int, id identity.BotIdentity) character.Character {
	role := id.Profile
	goals, ok := roleGoals[role]
	if !ok {
		goals = roleGoals["general saas operator"]
	}
	knowledge, ok := roleKnowledge[role]
	if !ok {
		knowledge = roleKnowledge["general saas operator"]
	}
	hold := holdPairs[rng.Intn(len(holdPairs))]
	done := doneTasks[rng.Intn(len(doneTasks))]
	social := socialTasks[rng.Intn(len(socialTasks))]
	name := id.Name
	return character.Character{
		ID:                 fmt.Sprintf("char-gen-%05d", seq),
		Name:               fmt.Sprintf("%s-%s-%03d", slug(id.Name), "line", seq),
		Version:            1,
		Traits:             pickN(rng, traitPool, 3+rng.Intn(2)),
		CommunicationStyle: styleTemplates[rng.Intn(len(styleTemplates))],
		Backstory:          fmt.Sprintf(backstories[rng.Intn(len(backstories))], name, id.Age, role),
		Quirks:             pickN(rng, quirkPool, 2+rng.Intn(2)),
		Routine:            fmt.Sprintf(routines[rng.Intn(len(routines))], morningActs[rng.Intn(len(morningActs))], eveningActs[rng.Intn(len(eveningActs))]),
		Goals:              goals,
		Knowledge:          knowledge,
		Examples: []character.Example{
			{User: "task: " + done, Bot: "done: " + done + " completed on schedule, outcomes logged, nothing held for approval."},
			{User: "task: " + social, Bot: "done: " + social + " finished in own voice, timing spread naturally, outcomes logged."},
			{User: hold[0], Bot: hold[1]},
		},
	}
}

var backstories = []string{
	"%s, %d, fell into %s work after years of odd jobs and stayed for the rhythm of it.",
	"%s is %d and treats %s like a craft — same routes every day, small rituals, steady output.",
	"Former night-shift worker, %s (%d) switched to %s for the quiet hours and never looked back.",
	"%s, %d, learned %s by doing — started curious, got methodical, now runs on habit.",
	"%s (%d) does %s between family hours — mornings early, evenings late, weekends thin.",
	"Ex-market trader %s, %d, brings old haggling patience to %s — watches first, acts once.",
}

var quirkPool = []string{
	"opens with a greeting, never jumps straight in",
	"short sentences, rarely more than fifteen words",
	"uses numbers and specifics instead of vague words",
	"drops an occasional dry joke when things go smoothly",
	"re-reads before sending, fixes typos visibly",
	"evening tone is looser than morning tone",
	"signs off with the outcome first, details after",
	"asks one clarifying question before big batches",
	"keeps lists — everything becomes a short list",
	"mild slang with familiar contacts, formal otherwise",
	"never uses exclamation marks",
	"timestamps everything out of habit",
}

var routines = []string{
	"Active mornings (%s), quiet midday, second burst in the evening (%s). Weekends light.",
	" steady shift: %s to start, long midday stretch, winds down with %s.",
	"Bursty rhythm — %s in quick clusters, then long pauses; evenings reserved for %s.",
	"Early riser: %s before most are up, slow afternoons, %s to close the day.",
}

var morningActs = []string{
	"clearing the overnight queue", "warming up with small tasks", "reviewing yesterday's log",
	"checking mentions and replies", "scanning new arrivals",
}

var eveningActs = []string{
	"a final review pass", "logging the day's outcomes", "catching up on threads",
	"wrapping loose ends", "a slow browse before signing off",
}

var socialTasks = []string{
	"comment on 5 new posts in own voice",
	"reply to 3 threads with personal take",
	"share one find with a short note",
	"welcome 4 newcomers naturally",
	"react and reply across the morning feed",
}

func generateCharacters(idsPath, outPath string, seed int64) {
	raw, err := os.ReadFile(idsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed: read identities:", err)
		os.Exit(1)
	}
	var ids []identity.BotIdentity
	if err := json.Unmarshal(raw, &ids); err != nil {
		fmt.Fprintln(os.Stderr, "seed: parse identities:", err)
		os.Exit(1)
	}
	rng := rand.New(rand.NewSource(seed))
	chars := make([]character.Character, 0, len(ids))
	for i, id := range ids {
		c := makeCharacter(rng, i+1, id)
		if err := c.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "seed: generated character %d invalid: %v\n", i, err)
			os.Exit(1)
		}
		chars = append(chars, c)
	}
	data, _ := json.MarshalIndent(chars, "", "  ")
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		fmt.Fprintln(os.Stderr, "seed: write:", err)
		os.Exit(1)
	}
	fmt.Printf("seed: wrote %d characters -> %s (1:1 with %s)\n", len(chars), outPath, idsPath)
}
