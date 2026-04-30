package audit

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

type stubExecer struct {
	calls int
	errs  []error
}

func (s *stubExecer) Exec(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
	s.calls++
	if len(s.errs) == 0 {
		return pgconn.CommandTag{}, nil
	}
	err := s.errs[0]
	s.errs = s.errs[1:]
	return pgconn.CommandTag{}, err
}

func TestRecord_RetriesTransientInsertFailure(t *testing.T) {
	db := &stubExecer{errs: []error{errors.New("temporary network failure"), nil}}
	service := &Service{db: db}

	err := service.Record(context.Background(), Event{
		Method: "POST",
		Path:   "/api/v1/test",
	})
	if err != nil {
		t.Fatalf("Record() error = %v, want nil", err)
	}
	if db.calls != 2 {
		t.Fatalf("Exec calls = %d, want 2", db.calls)
	}
}
