package api

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func writeInternalError(w http.ResponseWriter, err error) {
	writeError(w, http.StatusInternalServerError, err)
}

func publicError(status int, err error) string {
	if err == nil {
		return http.StatusText(status)
	}
	if status >= http.StatusInternalServerError {
		log.Printf("internal API error: %v", err)
		return "internal server error"
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return "resource not found"
	}
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) {
		log.Printf("database request error: code=%s constraint=%s", databaseError.Code, databaseError.ConstraintName)
		switch databaseError.Code {
		case "23505":
			return "resource already exists"
		case "23503":
			return "referenced resource does not exist or is still in use"
		default:
			return "request could not be processed"
		}
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "cipher") || strings.Contains(lower, "decrypt") || strings.Contains(lower, "sqlstate") {
		log.Printf("sensitive request error: %v", err)
		return "request could not be processed"
	}
	return err.Error()
}
