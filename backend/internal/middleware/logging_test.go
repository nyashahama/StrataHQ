package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogger_LogsRequestWithRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := RequestID(Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if buf.Len() == 0 {
		t.Fatal("expected log output, got empty")
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"request_id"`)) {
		t.Fatalf("log output does not include request_id: %s", buf.String())
	}
	if got := w.Header().Get(RequestIDHeader); got == "" {
		t.Fatal("response X-Request-ID header is empty")
	}
}
