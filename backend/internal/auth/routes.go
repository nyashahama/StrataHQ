package auth

import "github.com/go-chi/chi/v5"

// Routes registers public auth endpoints (no JWT required).
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)
	r.Post("/logout", h.Logout)
	r.Post("/forgot-password", h.ForgotPassword)
	r.Post("/reset-password", h.ResetPassword)
	return r
}

// OnboardingRoutes registers protected onboarding endpoints (requires JWT).
func (h *Handler) OnboardingRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/setup", h.Setup)
	return r
}

// ProtectedRoutes registers protected auth account endpoints (requires JWT).
func (h *Handler) ProtectedRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/me", h.Me)
	r.Patch("/profile", h.UpdateProfile)
	r.Patch("/org", h.UpdateOrg)
	r.Post("/change-password", h.ChangePassword)
	return r
}
