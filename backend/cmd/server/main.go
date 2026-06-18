package main

import (
	"context"
	"log"
	"net/http"
	"os"
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

	log.Printf("OpsCore API listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, server.Routes()); err != nil && err != http.ErrServerClosed {
		log.Println(err)
		os.Exit(1)
	}
}
