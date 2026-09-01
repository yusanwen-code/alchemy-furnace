package system

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alchemy-furnace/server/internal/engineendpoint"
	"github.com/gin-gonic/gin"
)

func TestDiagnosticsReportsHealthyPythonEngine(t *testing.T) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	engineendpoint.Set(server.URL)
	code, value, err := New(nil).Diagnostics(&gin.Context{})
	if err != nil {
		t.Fatalf("Diagnostics() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("Diagnostics() code = %v, want success", code)
	}
	if got := value.(*DiagnosticsResponse).PythonEngine; got != "ok" {
		t.Fatalf("PythonEngine = %q, want %q", got, "ok")
	}
}
