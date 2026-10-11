// Package api exposes Anarchy as an open, keyless service.
// Agents call POST /v1/bots and receive ready-to-run bot configs.
// Safety without auth: per-IP rate limit, hard count cap, and responses
// never include contact details (email/phone stay server-side).
package api

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
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

const (
	maxBotsPerRequest = 50
	requestsPerIP     = 10
	windowPerIP       = time.Hour
)

type server struct {
	pool *pgxpool.Pool
	key  []byte

	mu    sync.Mutex
	hits  map[string][]time.Time
}

func newServer(pool *pgxpool.Pool, key []byte) *server {
	return &server{pool: pool, key: key, hits: map[string][]time.Time{}}
}

// allow implements a fixed-window per-IP limiter. Single-instance only —
// fine for launch; swap for a shared store when we run replicas.
func (s *server) allow(ip string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	kept := s.hits[ip][:0]
	for _, t := range s.hits[ip] {
		if now.Sub(t) < windowPerIP {
			kept = append(kept, t)
		}
	}
	s.hits[ip] = kept
	if len(kept) >= requestsPerIP {
		return false
	}
	s.hits[ip] = append(kept, now)
	return true
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// publicBot is the agent-facing shape: full behavior + schedule,
// zero contact details. Email/phone never leave the server.
type publicBot struct {
	FormatVersion string              `json:"format_version"`
	DeploymentID  string              `json:"deployment_id"`
	IdentityName  string              `json:"identity_name"`
	IdentityAge   int                 `json:"identity_age,omitempty"`
	IdentityProfile string            `json:"identity_profile,omitempty"`
	Character     character.Character `json:"character"`
	Schedule      string              `json:"schedule"`
	JitterSeconds int                 `json:"jitter_seconds"`
	Permissions   integrations.Permissions `json:"permissions"`
}

type botsRequest struct {
	Count    int    `json:"count"`
	Schedule string `json:"schedule,omitempty"`
	Jitter   int    `json:"jitter_seconds,omitempty"`
	Seed     int64  `json:"seed,omitempty"`
}

type botsResponse struct {
	FormatVersion string      `json:"format_version"`
	Count         int         `json:"count"`
	Bots          []publicBot `json:"bots"`
	Rules         string      `json:"rules"`
}

func (s *server) handleBots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	if !s.allow(clientIP(r)) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit: 10 requests/hour per IP"})
		return
	}
	var req botsRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if req.Count < 1 {
		req.Count = 1
	}
	if req.Count > maxBotsPerRequest {
		req.Count = maxBotsPerRequest
	}
	schedule := req.Schedule
	if schedule == "" {
		schedule = "@every 2h"
	}
	if _, err := execution.ParseSchedule(schedule); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	jitter := req.Jitter
	if jitter == 0 {
		jitter = 600
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	ids, err := identity.ListActive(ctx, s.pool, s.key)
	if err != nil {
		log.Printf("api: identities: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	chars, err := character.ListAll(ctx, s.pool)
	if err != nil {
		log.Printf("api: characters: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	seed := req.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	deps, err := training.RandomPool(rand.New(rand.NewSource(seed)), ids, chars, req.Count)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "training pool empty; try later"})
		return
	}
	deps = training.AssignSchedule(deps, schedule, jitter)

	out := make([]publicBot, 0, len(deps))
	for _, d := range deps {
		out = append(out, publicBot{
			FormatVersion:   "anarchy-bot-v1",
			DeploymentID:    d.DeploymentID,
			IdentityName:    d.Identity.Name,
			IdentityAge:     d.Identity.Age,
			IdentityProfile: d.Identity.Profile,
			Character:       d.Character,
			Schedule:        d.Schedule,
			JitterSeconds:   d.JitterSeconds,
			Permissions:     integrations.DefaultPermissions(),
		})
	}
	writeJSON(w, http.StatusOK, botsResponse{
		FormatVersion: "anarchy-bot-v1",
		Count:         len(out),
		Bots:          out,
		Rules:         "Scheduled workers only. Hold task types in permissions.need_approval for the platform owner; disclose bot use where end users are affected. Max 50 bots per request, 10 requests/hour per IP.",
	})
}

func (s *server) handleEval(w http.ResponseWriter, r *http.Request) {
	if !s.allow(clientIP(r)) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	chars, err := character.ListAll(ctx, s.pool)
	if err != nil {
		log.Printf("api: characters: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	type version struct {
		CharacterID string `json:"character_id"`
		Version     int    `json:"version"`
		Examples    int    `json:"examples"`
		Passed      int    `json:"passed"`
	}
	versions := []version{}
	total, passed := 0, 0
	for _, c := range chars {
		v := version{CharacterID: c.ID, Version: c.Version, Examples: len(c.Examples)}
		for _, res := range evaluation.EvaluateCharacter(c) {
			total++
			if res.Passed {
				passed++
				v.Passed++
			}
		}
		versions = append(versions, v)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"total_examples": total,
		"passed":         passed,
		"failed":         total - passed,
		"versions":       versions,
	})
}

// Handler mounts the open API. DB creds come from env at boot; callers need nothing.
func Handler() (http.Handler, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := platform.Connect(ctx)
	if err != nil {
		return nil, err
	}
	key, err := platform.LoadContactKey()
	if err != nil {
		pool.Close()
		return nil, err
	}
	s := newServer(pool, key)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/v1/bots", s.handleBots)
	mux.HandleFunc("/v1/eval", s.handleEval)
	return mux, nil
}
