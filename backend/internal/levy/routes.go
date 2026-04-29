package levy

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/attention", h.PortfolioAttentionQueue)
	r.Get("/{schemeId}", h.Dashboard)
	r.Get("/{schemeId}/attention", h.AttentionQueue)
	r.Get("/{schemeId}/accounts/{accountId}/events", h.CollectionEvents)
	r.Post("/{schemeId}/accounts/{accountId}/events", h.RecordCollectionEvent)
	r.Get("/{schemeId}/accounts/{accountId}/reminder-draft", h.ReminderDraft)
	r.Post("/{schemeId}/accounts/{accountId}/reminders", h.SendReminder)
	r.Post("/{schemeId}/periods", h.CreatePeriod)
	r.Post("/{schemeId}/reconcile", h.Reconcile)
	r.Post("/{schemeId}/reconcile/imports", h.ImportBankStatement)
	r.Get("/{schemeId}/reconcile/imports/{importId}", h.GetBankStatementImport)
	r.Post("/{schemeId}/reconcile/imports/{importId}/apply", h.ApplyBankStatementImport)
	return r
}
