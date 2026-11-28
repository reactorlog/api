package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/reactorlog/api/internal/api"
	"github.com/reactorlog/api/internal/database"
)

func runServer(ctx context.Context) error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	handler, err := publicHandler(api.NewMux(api.NewStore(pool)))
	if err != nil {
		return err
	}
	handler = api.WithCORS(handler, api.ParseCORSOrigins(os.Getenv("CORS_ORIGINS")))
	return listenAndServe(ctx, httpAddr(), handler)
}

// publicHandler keeps root API mux behavior when STATIC_DIR is unset.
// With STATIC_DIR set, the SPA is served and the API is only under /api/*.
func publicHandler(apiMux http.Handler) (http.Handler, error) {
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		return apiMux, nil
	}
	static := os.DirFS(staticDir)
	return api.Public{API: apiMux, Static: static}, nil
}

func httpAddr() string {
	if addr := os.Getenv("HTTP_ADDR"); addr != "" {
		return addr
	}
	return ":8080"
}

func listenAndServe(ctx context.Context, addr string, handler http.Handler) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return serveHTTP(ctx, server)
}

func serveHTTP(ctx context.Context, server *http.Server) error {
	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", server.Addr)
		errCh <- server.ListenAndServe()
	}()
	return waitHTTP(ctx, server, errCh)
}

func waitHTTP(ctx context.Context, server *http.Server, errCh <-chan error) error {
	select {
	case <-ctx.Done():
		return shutdownHTTP(server, errCh)
	case err := <-errCh:
		return normalizeServeErr(err)
	}
}

func shutdownHTTP(server *http.Server, errCh <-chan error) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	return normalizeServeErr(<-errCh)
}

func normalizeServeErr(err error) error {
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
