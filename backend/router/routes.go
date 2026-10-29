package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	_ "modernc.org/sqlite"

	"backup-monitor-backend/router/auth"
	"backup-monitor-backend/router/common"
	mdmw "backup-monitor-backend/router/middleware"
	"backup-monitor-backend/router/settings"
	"backup-monitor-backend/router/swagger"
	"backup-monitor-backend/router/system"
)

func (a *App) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(mdmw.CORS(a.cfg.CORSOrigins))
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	timeout := chimw.Timeout(60 * time.Second)
	r.Use(func(next http.Handler) http.Handler {
		timed := timeout(next)
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/api/agent/ws" {
				next.ServeHTTP(w, req)
				return
			}
			timed.ServeHTTP(w, req)
		})
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		common.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		common.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Mount("/swagger", swagger.Handler())

	a.settingsR = settings.New(a.db, a.audit, a.userWorkflow, a.cfg)
	a.authR = auth.New(a.db, a.audit, a.authWorkflow, a.cfg)
	a.systemR = system.New(a.db, a.audit, a.cfg)

	a.authR.RegisterPublic(r)
	a.settingsR.RegisterPublic(r)
	a.backupR.RegisterPublic(r)
	r.Group(func(r chi.Router) {
		r.Use(mdmw.Auth(a.cfg.JWTSecret, a.db))
		a.authR.Register(r)

		requirePermission := func(permission string) func(http.Handler) http.Handler {
			return mdmw.Permission(a.db, permission)
		}
		a.systemR.Register(r, requirePermission)
		a.settingsR.Register(r, requirePermission)
		a.backupR.Register(r, requirePermission)
	})

	return r
}
