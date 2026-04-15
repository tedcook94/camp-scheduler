package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

func newSPATestServer(staticFS *fstest.MapFS) *Server {
	gin.SetMode(gin.TestMode)
	s := &Server{
		router:   gin.New(),
		staticFS: staticFS,
	}
	// Register a real route so NoRoute only fires for unmatched paths
	s.router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	s.serveSPA()
	return s
}

func testFS() *fstest.MapFS {
	return &fstest.MapFS{
		"index.html":        {Data: []byte("<html>SPA</html>")},
		"_app/immutable.js": {Data: []byte("console.log('app')")},
		"assets/style.css":  {Data: []byte("body{}")},
	}
}

func doRequest(s *Server, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	s.router.ServeHTTP(w, req)
	return w
}

func TestSPA_AdminRoute_ServesIndexHTML(t *testing.T) {
	s := newSPATestServer(testFS())

	tests := []struct {
		name string
		path string
	}{
		{"exact /admin", "/admin"},
		{"trailing slash", "/admin/"},
		{"nested route", "/admin/camps"},
		{"deep nested route", "/admin/users/some-id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doRequest(s, "GET", tt.path)
			if w.Code != http.StatusOK {
				t.Errorf("GET %s: status = %d, want %d", tt.path, w.Code, http.StatusOK)
			}
			if body := w.Body.String(); body != "<html>SPA</html>" {
				t.Errorf("GET %s: body = %q, want index.html content", tt.path, body)
			}
		})
	}
}

func TestSPA_StaticAsset_ServesFile(t *testing.T) {
	s := newSPATestServer(testFS())

	tests := []struct {
		name        string
		path        string
		wantBody    string
		wantContent string
	}{
		{"js asset", "/admin/_app/immutable.js", "console.log('app')", "text/javascript"},
		{"css asset", "/admin/assets/style.css", "body{}", "text/css"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doRequest(s, "GET", tt.path)
			if w.Code != http.StatusOK {
				t.Errorf("GET %s: status = %d, want %d", tt.path, w.Code, http.StatusOK)
			}
			if body := w.Body.String(); body != tt.wantBody {
				t.Errorf("GET %s: body = %q, want %q", tt.path, body, tt.wantBody)
			}
		})
	}
}

func TestSPA_MissingAsset_Returns404(t *testing.T) {
	s := newSPATestServer(testFS())

	tests := []struct {
		name string
		path string
	}{
		{"missing js", "/admin/_app/missing.js"},
		{"missing css", "/admin/assets/missing.css"},
		{"missing image", "/admin/favicon.png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doRequest(s, "GET", tt.path)
			if w.Code != http.StatusNotFound {
				t.Errorf("GET %s: status = %d, want %d", tt.path, w.Code, http.StatusNotFound)
			}
		})
	}
}

func TestSPA_NonAdminPath_Returns404(t *testing.T) {
	s := newSPATestServer(testFS())

	tests := []struct {
		name string
		path string
	}{
		{"root", "/"},
		{"random path", "/foo"},
		{"adminfoo (no slash)", "/adminfoo"},
		{"adminpanel", "/adminpanel/stuff"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doRequest(s, "GET", tt.path)
			if w.Code != http.StatusNotFound {
				t.Errorf("GET %s: status = %d, want %d", tt.path, w.Code, http.StatusNotFound)
			}
		})
	}
}

func TestSPA_DirectoryPath_FallsBackToIndex(t *testing.T) {
	s := newSPATestServer(testFS())

	// /admin/_app/ is a directory in the FS — should fall back to index.html,
	// not expose a directory listing
	tests := []struct {
		name string
		path string
	}{
		{"_app directory", "/admin/_app/"},
		{"assets directory", "/admin/assets/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doRequest(s, "GET", tt.path)
			if w.Code != http.StatusOK {
				t.Errorf("GET %s: status = %d, want %d", tt.path, w.Code, http.StatusOK)
			}
			if body := w.Body.String(); body != "<html>SPA</html>" {
				t.Errorf("GET %s: body = %q, want index.html content", tt.path, body)
			}
		})
	}
}

func TestSPA_APIRoute_NotIntercepted(t *testing.T) {
	s := newSPATestServer(testFS())

	w := doRequest(s, "GET", "/api/v1/test")
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/v1/test: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSPA_NilStaticFS_NoRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{
		router:   gin.New(),
		staticFS: nil,
	}
	// Don't call serveSPA — simulate what routes() does when staticFS is nil
	s.router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Without NoRoute registered, Gin returns 404 for unknown paths
	w := doRequest(s, "GET", "/admin")
	if w.Code != http.StatusNotFound {
		t.Errorf("GET /admin with nil staticFS: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
