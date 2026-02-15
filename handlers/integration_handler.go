package handlers

import (
	"net/http"

	"github.com/gorilla/mux"

	"go-microservice/services"
)

type IntegrationHandler struct {
	integration *services.IntegrationService
}

func NewIntegrationHandler(integration *services.IntegrationService) *IntegrationHandler {
	return &IntegrationHandler{integration: integration}
}

func (h *IntegrationHandler) Register(r *mux.Router) {
	r.HandleFunc("/integration/health", h.Health).Methods(http.MethodGet)
}

func (h *IntegrationHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"minio_enabled": h.integration.Enabled(),
	})
}
