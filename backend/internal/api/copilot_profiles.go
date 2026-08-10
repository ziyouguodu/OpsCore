package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"opscore/backend/internal/models"
)

func (s *Server) copilotProfiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.store.ListCopilotModelConfigs(r.Context())
		if err != nil {
			writeInternalError(w, err)
			return
		}
		for index := range items {
			items[index].APIKey = ""
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var item models.CopilotModelConfig
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		normalizeCopilotModelConfigInput(&item)
		if err := validateCopilotModelConfig(item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		created, err := s.store.CreateCopilotModelConfig(r.Context(), item)
		if err != nil {
			writeCopilotProfileError(w, err)
			return
		}
		created.APIKey = ""
		writeJSON(w, http.StatusCreated, created)
	}
}

func (s *Server) copilotProfileResource(w http.ResponseWriter, r *http.Request) {
	id, ok := copilotProfileID(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodPut:
		var item models.CopilotModelConfig
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		normalizeCopilotModelConfigInput(&item)
		if err := validateCopilotModelConfig(item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		updated, err := s.store.UpdateCopilotModelConfig(r.Context(), id, item)
		if err != nil {
			writeCopilotProfileError(w, err)
			return
		}
		updated.APIKey = ""
		writeJSON(w, http.StatusOK, updated)
	case http.MethodDelete:
		if err := s.store.DeleteCopilotModelConfig(r.Context(), id); err != nil {
			writeCopilotProfileError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) copilotProfileActivate(w http.ResponseWriter, r *http.Request) {
	id, ok := copilotProfileID(w, r)
	if !ok {
		return
	}
	item, err := s.store.ActivateCopilotModelConfig(r.Context(), id)
	if err != nil {
		writeCopilotProfileError(w, err)
		return
	}
	item.APIKey = ""
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) copilotProfileTestConnection(w http.ResponseWriter, r *http.Request) {
	id, ok := copilotProfileID(w, r)
	if !ok {
		return
	}
	item, err := s.store.GetCopilotModelConfig(r.Context(), id)
	if err != nil {
		writeCopilotProfileError(w, err)
		return
	}
	key, err := s.store.GetCopilotModelAPIKey(r.Context(), id)
	if err != nil {
		writeCopilotProfileError(w, err)
		return
	}
	result, err := testCopilotConnection(r.Context(), copilotConnectionRequest{
		Provider: item.Provider, Endpoint: item.Endpoint, Model: item.Model, APIKey: key,
		LocalEndpoint: item.LocalEndpoint, LocalModel: item.LocalModel,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func copilotProfileID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, errors.New("invalid copilot model configuration id"))
		return 0, false
	}
	return id, true
}

func normalizeCopilotModelConfigInput(item *models.CopilotModelConfig) {
	item.Name = strings.TrimSpace(item.Name)
	item.Provider = normalizeCopilotProvider(item.Provider)
	item.Temperature = strings.TrimSpace(item.Temperature)
	item.MaxTokens = strings.TrimSpace(item.MaxTokens)
	if item.Temperature == "" {
		item.Temperature = "0.2"
	}
	if item.MaxTokens == "" {
		item.MaxTokens = "2048"
	}
	item.AuditEnabled = true
	if item.Provider == "local" {
		item.Endpoint = ""
		item.Model = ""
		item.APIKey = ""
	} else {
		item.LocalEndpoint = ""
		item.LocalModel = ""
	}
}

func validateCopilotModelConfig(item models.CopilotModelConfig) error {
	if item.Name == "" || len([]rune(item.Name)) > 80 {
		return errors.New("name is required and must not exceed 80 characters")
	}
	return validateCopilotConfig(models.CopilotConfig{
		Provider: item.Provider, Endpoint: item.Endpoint, Model: item.Model,
		LocalEndpoint: item.LocalEndpoint, LocalModel: item.LocalModel,
		Temperature: item.Temperature, MaxTokens: item.MaxTokens,
	})
}

func writeCopilotProfileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errCopilotProfileNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, errCopilotActiveProfileDelete), errors.Is(err, errCopilotProfileKeyRequired):
		writeError(w, http.StatusConflict, err)
	default:
		var databaseError *pgconn.PgError
		if errors.As(err, &databaseError) && databaseError.Code == "23505" {
			writeError(w, http.StatusConflict, errors.New("copilot model configuration name already exists"))
			return
		}
		writeInternalError(w, err)
	}
}
