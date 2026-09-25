package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"opscore/backend/internal/models"
)

func parseListQuery(r *http.Request) (models.ListQuery, error) {
	values := r.URL.Query()
	page, err := positiveQueryInt(values.Get("page"), 1)
	if err != nil {
		return models.ListQuery{}, errors.New("page must be a positive integer")
	}
	pageSize, err := positiveQueryInt(values.Get("pageSize"), 20)
	if err != nil || pageSize > 100 {
		return models.ListQuery{}, errors.New("pageSize must be between 1 and 100")
	}
	order := strings.ToLower(strings.TrimSpace(values.Get("order")))
	if order != "" && order != "asc" && order != "desc" {
		return models.ListQuery{}, errors.New("order must be asc or desc")
	}
	return models.ListQuery{
		Page:        page,
		PageSize:    pageSize,
		Keyword:     values.Get("keyword"),
		IPs:         parseIPList(values.Get("ips")),
		Type:        values.Get("type"),
		Kind:        values.Get("kind"),
		Environment: values.Get("environment"),
		Business:    values.Get("business"),
		NetworkZone: values.Get("networkZone"),
		Status:      values.Get("status"),
		Level:       values.Get("level"),
		Sort:        values.Get("sort"),
		Order:       order,
	}, nil
}

func parseIPList(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', '，', ';', '；', '、', '\n', '\r', '\t', ' ':
			return true
		default:
			return false
		}
	})
	seen := make(map[string]struct{}, len(fields))
	ips := make([]string, 0, len(fields))
	for _, field := range fields {
		ip := strings.ToLower(strings.TrimSpace(field))
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		ips = append(ips, ip)
	}
	return ips
}

func positiveQueryInt(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, errors.New("value must be a positive integer")
	}
	return value, nil
}
