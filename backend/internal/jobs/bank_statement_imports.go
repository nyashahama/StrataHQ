package jobs

import (
	"context"
	"encoding/json"
	"fmt"
)

type BankStatementImportPayload struct {
	ImportID string `json:"importId"`
}

type importProcessor interface {
	ProcessBankStatementImport(ctx context.Context, importID string) error
}

type BankStatementImportHandler struct {
	levyService importProcessor
}

func NewBankStatementImportHandler(service importProcessor) *BankStatementImportHandler {
	return &BankStatementImportHandler{levyService: service}
}

func (h *BankStatementImportHandler) Handle(ctx context.Context, payload json.RawMessage) error {
	var p BankStatementImportPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("%w: invalid bank statement import payload: %v", ErrNonRetryable, err)
	}
	if p.ImportID == "" {
		return fmt.Errorf("%w: importId is required", ErrNonRetryable)
	}
	return h.levyService.ProcessBankStatementImport(ctx, p.ImportID)
}
