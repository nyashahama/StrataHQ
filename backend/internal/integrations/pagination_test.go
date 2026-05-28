package integrations

import (
	"errors"
	"net/http/httptest"
	"testing"
)

func TestParsePaginationCapsPerPage(t *testing.T) {
	req := httptest.NewRequest("GET", "/openapi/schemes/scheme/levy-accounts?page=2&per_page=500", nil)

	page, limitRows, offsetRows, err := parsePagination(req)

	if err != nil {
		t.Fatalf("parsePagination returned error: %v", err)
	}
	if page != 2 {
		t.Fatalf("page = %d, want 2", page)
	}
	if limitRows != 200 {
		t.Fatalf("limitRows = %d, want 200", limitRows)
	}
	if offsetRows != 200 {
		t.Fatalf("offsetRows = %d, want 200", offsetRows)
	}
}

func TestParsePaginationRejectsOffsetOverflow(t *testing.T) {
	req := httptest.NewRequest("GET", "/openapi/schemes/scheme/levy-accounts?page=42949674&per_page=50", nil)

	_, _, _, err := parsePagination(req)

	if !errors.Is(err, errPaginationOutOfRange) {
		t.Fatalf("error = %v, want %v", err, errPaginationOutOfRange)
	}
}

func TestParsePaginationAllowsMaxInt32Offset(t *testing.T) {
	req := httptest.NewRequest("GET", "/openapi/schemes/scheme/levy-accounts?page=10737419&per_page=200", nil)

	page, limitRows, offsetRows, err := parsePagination(req)

	if err != nil {
		t.Fatalf("parsePagination returned error: %v", err)
	}
	if page != 10737419 {
		t.Fatalf("page = %d, want 10737419", page)
	}
	if limitRows != 200 {
		t.Fatalf("limitRows = %d, want 200", limitRows)
	}
	if offsetRows != 2147483600 {
		t.Fatalf("offsetRows = %d, want 2147483600", offsetRows)
	}
}
