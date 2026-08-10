package store

import (
	"testing"

	"opscore/backend/internal/models"
)

func TestClampListPageReturnsLastAvailablePage(t *testing.T) {
	query := models.ListQuery{Page: 9, PageSize: 20}
	pages := clampListPage(&query, 41)
	if pages != 3 || query.Page != 3 {
		t.Fatalf("expected page 3 of 3, got page %d of %d", query.Page, pages)
	}
}

func TestClampListPageKeepsEmptyListsOnFirstPage(t *testing.T) {
	query := models.ListQuery{Page: 4, PageSize: 20}
	pages := clampListPage(&query, 0)
	if pages != 1 || query.Page != 1 {
		t.Fatalf("expected empty list page 1 of 1, got page %d of %d", query.Page, pages)
	}
}
