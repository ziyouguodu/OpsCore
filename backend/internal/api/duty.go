package api

import (
	"errors"
	"net/http"

	"opscore/backend/internal/models"
)

func (s *Server) dutyCenter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		state, err := s.store.GetDutyCenter(r.Context())
		if err != nil {
			writeInternalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, state)
	case http.MethodPut:
		var mutation models.DutyCenterMutation
		if err := readJSON(r, &mutation); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		claims := claimsFrom(r.Context())
		saved, err := s.store.SaveDutyCenter(r.Context(), mutation, claims.UserID)
		if err != nil {
			if errors.Is(err, errDutyRevisionConflict) {
				writeError(w, http.StatusConflict, err)
				return
			}
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, saved)
	}
}
