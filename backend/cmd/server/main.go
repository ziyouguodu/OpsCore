package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"opscore/backend/internal/api"
	"opscore/backend/internal/auth"
	"opscore/backend/internal/config"
	secretcrypto "opscore/backend/internal/crypto"
	"opscore/backend/internal/store"
)

func main() {
	cfg := config.FromEnv()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	credentialBox, err := secretcrypto.NewSecretBox(cfg.CredentialKey)
	if err != nil {
		log.Fatalf("credential encryption: %v", err)
	}

	db, err := store.Open(ctx, cfg.DatabaseURL, credentialBox)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := db.SeedDefaults(ctx, cfg.InitialAdminPassword); err != nil {
		log.Fatalf("seed defaults: %v", err)
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "reset-admin-password":
			resetPassword := os.Getenv("OPSCORE_RESET_ADMIN_PASSWORD")
			if resetPassword == "" {
				log.Fatal("OPSCORE_RESET_ADMIN_PASSWORD is required")
			}
			user, err := db.ResetUserPassword(ctx, "admin", resetPassword, true)
			if err != nil {
				log.Fatalf("reset admin password: %v", err)
			}
			log.Printf("admin password reset for %s; mustChangePassword=true", user.Username)
			return
		default:
			log.Fatalf("unknown command %q", os.Args[1])
		}
	}

	signer := auth.NewSigner(cfg.JWTSecret, 24*time.Hour)
	server := api.NewServer(db, signer, cfg)

	httpServer := newHTTPServer(cfg.ListenAddr, server.Routes())
	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("OpsCore API listening on %s", cfg.ListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("serve API: %v", err)
			stop()
		}
	}()

	<-runCtx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown API: %v", err)
	}
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
