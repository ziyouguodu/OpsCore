package api

import (
	"errors"
	"net/http"

	"opscore/backend/internal/domain"
	"opscore/backend/internal/models"
)

func (s *Server) tasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		query, err := parseListQuery(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		items, err := s.store.ListTasksPage(r.Context(), query)
		if err != nil {
			writeInternalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var item models.Task
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if item.Status == "" {
			item.Status = string(domain.TaskPending)
		}
		if item.Type == "" {
			item.Type = "任务"
		}
		if err := validateTaskMutation(item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateTaskStatus(item.Status); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if item.Status != string(domain.TaskPending) {
			writeError(w, http.StatusBadRequest, errors.New("new task status must be 待处理"))
			return
		}
		saved, err := s.store.CreateTask(r.Context(), item)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, saved)
	}
}

func (s *Server) taskResource(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var item models.Task
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if item.Status == "" {
			item.Status = string(domain.TaskPending)
		}
		if item.Type == "" {
			item.Type = "任务"
		}
		if err := validateTaskMutation(item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateTaskStatus(item.Status); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		current, err := s.store.GetTaskStatus(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		if err := validateTaskTransition(current, item.Status); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
		saved, err := s.store.UpdateTask(r.Context(), id, item)
		if err != nil {
			if errors.Is(err, errStatusConflict) {
				writeError(w, http.StatusConflict, err)
				return
			}
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, saved)
	case http.MethodDelete:
		if err := s.store.DeleteTask(r.Context(), id); err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) taskStatus(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateTaskStatus(body.Status); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	current, err := s.store.GetTaskStatus(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err := validateTaskTransition(current, body.Status); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	if err := s.store.UpdateTaskStatus(r.Context(), id, body.Status); err != nil {
		if errors.Is(err, errStatusConflict) {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": body.Status})
}

func (s *Server) incidents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		query, err := parseListQuery(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		items, err := s.store.ListIncidentsPage(r.Context(), query)
		if err != nil {
			writeInternalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var item models.Incident
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if item.Status == "" {
			item.Status = string(domain.IncidentNew)
		}
		if item.Level == "" {
			item.Level = "P3"
		}
		if err := validateIncidentMutation(item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateIncidentLevel(item.Level); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateIncidentStatus(item.Status); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if item.Status != string(domain.IncidentNew) {
			writeError(w, http.StatusBadRequest, errors.New("new incident status must be 新建"))
			return
		}
		saved, err := s.store.CreateIncident(r.Context(), item)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, saved)
	}
}

func (s *Server) incidentResource(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var item models.Incident
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if item.Status == "" {
			item.Status = string(domain.IncidentNew)
		}
		if item.Level == "" {
			item.Level = "P3"
		}
		if err := validateIncidentMutation(item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateIncidentLevel(item.Level); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateIncidentStatus(item.Status); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		current, err := s.store.GetIncidentStatus(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		if err := validateIncidentTransition(current, item.Status); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
		saved, err := s.store.UpdateIncident(r.Context(), id, item)
		if err != nil {
			if errors.Is(err, errStatusConflict) {
				writeError(w, http.StatusConflict, err)
				return
			}
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, saved)
	case http.MethodDelete:
		if err := s.store.DeleteIncident(r.Context(), id); err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) incidentStatus(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateIncidentStatus(body.Status); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	current, err := s.store.GetIncidentStatus(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err := validateIncidentTransition(current, body.Status); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	if err := s.store.UpdateIncidentStatus(r.Context(), id, body.Status); err != nil {
		if errors.Is(err, errStatusConflict) {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": body.Status})
}
