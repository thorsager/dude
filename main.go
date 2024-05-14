package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thorsager/dude/dude"
	"github.com/thorsager/dude/metricsandlogging"
	"github.com/thorsager/dude/middleware"
	"github.com/thorsager/dude/persistence"
	"github.com/thorsager/dude/requestid"
)

const EnvDbPrefix = "DB_"
const EnvUrlSuffix = "_URL"

var defaultDataBase = ""
var BindAddress = ":7070"

var middleWareHandler = middleware.Compose(
	persistence.Middleware(dbSelector),
	metricsandlogging.Middleware,
	requestid.Middleware,
)

func dbSelector(r *http.Request) string {
	if s := r.Header.Get("X-DB-Name"); s != "" {
		return strings.ToUpper(s)
	}
	return defaultDataBase
}

func readEnvironment() ([]persistence.NamedUrl, error) {
	var nurls []persistence.NamedUrl
	for _, envKv := range os.Environ() {
		if strings.HasPrefix(envKv, EnvDbPrefix) {
			kv := strings.SplitN(envKv, "=", 2)
			if !strings.HasSuffix(kv[0], EnvUrlSuffix) {
				continue
			}
			name := strings.TrimPrefix(kv[0], EnvDbPrefix)
			name = strings.TrimSuffix(name, EnvUrlSuffix)
			u, err := url.Parse(kv[1])
			if err != nil {
				return nurls, err
			}
			nurls = append(nurls, persistence.NamedUrl{Name: strings.ToUpper(name), Url: u})
		}
	}
	return nurls, nil
}

func main() {
	flag.StringVar(&BindAddress, "addr", ":8080", "Bind Address")
	flag.Parse()

	// handle SIGINT
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Fatal("Error loading .env file")
	}

	nurls, err := readEnvironment()
	if err != nil {
		panic(err)
	}

	err = persistence.Setup(nurls)
	if err != nil {
		persistence.Close()
		panic(err)
	}
	defer persistence.Close()

	// If only one database is configured, use it as the default
	if len(nurls) == 1 {
		defaultDataBase = nurls[0].Name
	}

	serveMux := http.NewServeMux()

	// Register the metrics handler
	serveMux.Handle("/metrics", promhttp.Handler())

	serveMux.HandleFunc("POST /dude", dude.Create)
	serveMux.HandleFunc("GET /dude", dude.GetAll)
	serveMux.HandleFunc("GET /dude/{id}", dude.GetById)
	serveMux.HandleFunc("PUT /dude", dude.Update)
	serveMux.HandleFunc("DELETE /dude/{id}", dude.Delete)

	srv := &http.Server{
		Addr:        BindAddress,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
		Handler:     middleWareHandler(serveMux),
	}

	// Start the server
	srvErr := make(chan error, 1)
	go func() {
		log.Printf("Server starting on port %s", BindAddress)
		srvErr <- srv.ListenAndServe()
	}()

	// Wait for either an error or a signal
	select {
	case err = <-srvErr:
		log.Fatalf("while starting: %s", err)
		return
	case <-ctx.Done():
		stop() // stop listening for SIGINT
	}

	// Shutdown the server
	log.Printf("Shutting down server")
	err = srv.Shutdown(context.Background())
	if err != nil {
		log.Fatalf("while shutting down: %s", err)
	}
}
