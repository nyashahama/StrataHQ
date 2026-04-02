package agm

import "github.com/go-chi/chi/v5"

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{schemeId}", h.Dashboard)
	r.Post("/{schemeId}/meetings", h.ScheduleMeeting)
	r.Post("/{schemeId}/meetings/{meetingId}/proxy", h.AssignProxy)
	r.Post("/{schemeId}/resolutions/{resolutionId}/vote", h.CastVote)
	return r
}
