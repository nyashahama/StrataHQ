package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type fakeImportProcessor struct {
	importID string
	err      error
}

func (f *fakeImportProcessor) ProcessBankStatementImport(ctx context.Context, importID string) error {
	f.importID = importID
	return f.err
}

func TestBankStatementImportHandlerReturnsErrorOnParseFailure(t *testing.T) {
	svc := &fakeImportProcessor{err: errors.New("temporary database failure")}
	handler := NewBankStatementImportHandler(svc)
	err := handler.Handle(context.Background(), json.RawMessage(`{"importId":"import-1"}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBankStatementImportHandlerRejectsMalformedPayload(t *testing.T) {
	svc := &fakeImportProcessor{}
	handler := NewBankStatementImportHandler(svc)
	err := handler.Handle(context.Background(), json.RawMessage(`{invalid`))
	if !errors.Is(err, ErrBadPayload) {
		t.Fatalf("expected ErrBadPayload, got: %v", err)
	}
}

func TestBankStatementImportHandlerCallsLevyService(t *testing.T) {
	svc := &fakeImportProcessor{}
	handler := NewBankStatementImportHandler(svc)
	if err := handler.Handle(context.Background(), json.RawMessage(`{"importId":"import-1"}`)); err != nil {
		t.Fatalf("handle job: %v", err)
	}
	if svc.importID != "import-1" {
		t.Fatalf("importID = %q, want import-1", svc.importID)
	}
}

func TestBankStatementImportHandlerRejectsEmptyImportID(t *testing.T) {
	svc := &fakeImportProcessor{}
	handler := NewBankStatementImportHandler(svc)
	err := handler.Handle(context.Background(), json.RawMessage(`{"importId":""}`))
	if !errors.Is(err, ErrBadPayload) {
		t.Fatalf("expected ErrBadPayload, got: %v", err)
	}
}
