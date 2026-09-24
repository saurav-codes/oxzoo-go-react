package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	port := flag.Int("port", 9113, "listen port")
	flag.Parse()

	ctx := context.Background()

	// ox injects DATABASE_URL/REDIS_URL from the declared services; the
	// fallbacks keep local `go run` usable without the dashboard env editor.
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres@127.0.0.1:5432/oxzoo_go_react"
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://127.0.0.1:6379/0"
	}
	redisOpt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal(err)
	}
	rdb := redis.NewClient(redisOpt)
	defer rdb.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/greeting", func(w http.ResponseWriter, r *http.Request) {
		line := fmt.Sprintf("hello world oxzoo-go-react_%s", os.Getenv("GREETING_TAG"))
		// Every hit is logged to postgres; /api/stats reports the running count.
		if _, err := pool.Exec(ctx, "INSERT INTO greeting_log (greeting) VALUES ($1)", line); err != nil {
			http.Error(w, "greeting_log insert: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, line)
	})
	mux.HandleFunc("/api/visits", func(w http.ResponseWriter, r *http.Request) {
		// INCR is atomic; the TTL is set on the first hit so the counter expires.
		count, err := rdb.Incr(ctx, "oxzoo:visits").Result()
		if err != nil {
			http.Error(w, "redis incr: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if count == 1 {
			rdb.Expire(ctx, "oxzoo:visits", 3600*time.Second)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"visits":%d}`, count)
	})
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		var count int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM greeting_log").Scan(&count); err != nil {
			http.Error(w, "greeting_log count: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"greetings_logged":%d}`, count)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "ok")
	})

	if err := http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", *port), mux); err != nil {
		fmt.Fprintln(os.Stderr, "server error:", err)
		os.Exit(1)
	}
}
