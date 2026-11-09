package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
)

type auditLogWriter interface {
	CreateAuditLog(context.Context, dbgen.CreateAuditLogParams) error
}

// AuditMutations menyimpan konteks mutasi API yang berhasil tanpa menyimpan
// body request, yang mungkin berisi data sensitif.
func AuditMutations(writer auditLogWriter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isMutation(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			if rec.status < http.StatusOK || rec.status >= http.StatusMultipleChoices {
				return
			}
			identity, ok := auth.IdentityFrom(r.Context())
			if !ok {
				return
			}

			path := r.URL.Path
			if len(path) > 2048 {
				path = path[:2048]
			}
			ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 2*time.Second)
			defer cancel()
			if err := writer.CreateAuditLog(ctx, dbgen.CreateAuditLogParams{
				ActorID: identity.UserID, HttpMethod: r.Method, RequestPath: path,
				StatusCode: int16(rec.status), RequestID: httputil.RequestIDFromContext(r.Context()),
			}); err != nil {
				slog.ErrorContext(ctx, "gagal menyimpan audit log",
					slog.Int64("actor_id", identity.UserID), slog.String("path", path), slog.Any("error", err))
			}
		})
	}
}

func isMutation(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
