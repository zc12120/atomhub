package app

import (
	"net/http"

	"github.com/zc12120/atomhub/internal/auth"
)

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", a.handleHealthz)
	mux.HandleFunc("POST /admin/login", a.handleAdminLogin)
	mux.HandleFunc("POST /admin/logout", a.handleAdminLogout)
	mux.HandleFunc("GET /admin/session", a.handleAdminSession)

	adminMux := http.NewServeMux()
	adminMux.HandleFunc("GET /admin/me", a.handleAdminMe)
	adminMux.HandleFunc("GET /admin/dashboard", a.handleDashboard)
	adminMux.HandleFunc("GET /admin/keys", a.handleListKeys)
	adminMux.HandleFunc("POST /admin/keys", a.handleCreateKey)
	adminMux.HandleFunc("PUT /admin/keys/{id}", a.handleUpdateKey)
	adminMux.HandleFunc("DELETE /admin/keys/{id}", a.handleDeleteKey)
	adminMux.HandleFunc("POST /admin/keys/{id}/probe", a.handleProbeKey)
	adminMux.HandleFunc("GET /admin/models", a.handleModels)
	adminMux.HandleFunc("GET /admin/health", a.handleHealth)
	adminMux.HandleFunc("GET /admin/requests", a.handleRequests)
	adminMux.HandleFunc("GET /admin/downstream-keys", a.handleListDownstreamKeys)
	adminMux.HandleFunc("POST /admin/downstream-keys", a.handleCreateDownstreamKey)
	adminMux.HandleFunc("PUT /admin/downstream-keys/{id}", a.handleUpdateDownstreamKey)
	adminMux.HandleFunc("DELETE /admin/downstream-keys/{id}", a.handleDeleteDownstreamKey)
	adminMux.HandleFunc("GET /admin/downstream-keys/{id}/token", a.handleRevealDownstreamKey)
	adminMux.HandleFunc("POST /admin/downstream-keys/{id}/regenerate", a.handleRegenerateDownstreamKey)

	mux.Handle("/admin/", auth.RequireAdmin(a.sessionManager, adminMux))

	gatewayMux := http.NewServeMux()
	gatewayMux.HandleFunc("GET /v1/models", a.handleGatewayModels)
	gatewayMux.HandleFunc("POST /v1/chat/completions", a.handleChatCompletions)
	mux.Handle("/v1/", a.requireGatewayAccess(gatewayMux))

	mux.Handle("/", spaHandler("web/dist"))
	return mux
}
