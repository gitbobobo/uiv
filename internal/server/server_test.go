package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gitbobobo/uiv/internal/config"
	"github.com/gitbobobo/uiv/internal/store"
)

const token = "test-token"

var pngBytes = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 100)...)

func newTestServer(t *testing.T, cfg config.Config) http.Handler {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	cfg.Token = token
	if cfg.MaxSize == 0 {
		cfg.MaxSize = 1 << 20
	}
	static := fstest.MapFS{"index.html": {Data: []byte("<html>uiv</html>")}}
	return New(cfg, st, static, "v-test")
}

func uploadReq(t *testing.T, filename string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", filename)
	fw.Write(content)
	mw.Close()
	req := httptest.NewRequest("POST", "http://nas.example:8080/api/files", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func do(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestUploadServeListDelete(t *testing.T) {
	h := newTestServer(t, config.Config{})

	rec := do(h, uploadReq(t, `C:\shots\login [v2].png`, pngBytes))
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload status %d: %s", rec.Code, rec.Body)
	}
	var f fileJSON
	json.Unmarshal(rec.Body.Bytes(), &f)
	wantURL := "http://nas.example:8080/f/" + f.ID + ".png"
	if f.URL != wantURL || f.Name != "login [v2].png" || f.Type != "image/png" || f.Size != int64(len(pngBytes)) {
		t.Fatalf("unexpected upload response: %+v", f)
	}
	if f.Markdown != `![login \[v2\].png](`+wantURL+`)` {
		t.Fatalf("markdown = %q", f.Markdown)
	}

	rec = do(h, httptest.NewRequest("GET", "/f/"+f.ID+".png", nil))
	if rec.Code != 200 || !bytes.Equal(rec.Body.Bytes(), pngBytes) {
		t.Fatalf("serve status %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "image/png" || rec.Header().Get("X-Content-Type-Options") != "nosniff" ||
		!strings.Contains(rec.Header().Get("Cache-Control"), "immutable") || rec.Header().Get("Content-Security-Policy") != "" {
		t.Fatalf("unexpected headers: %v", rec.Header())
	}
	if rec := do(h, httptest.NewRequest("GET", "/f/"+f.ID+".jpg", nil)); rec.Code != 404 {
		t.Fatalf("wrong extension should 404, got %d", rec.Code)
	}

	req := httptest.NewRequest("GET", "/api/files", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = do(h, req)
	var list struct {
		Files      []fileJSON `json:"files"`
		NextCursor string     `json:"next_cursor"`
		BaseURL    string     `json:"base_url"`
	}
	json.Unmarshal(rec.Body.Bytes(), &list)
	if rec.Code != 200 || len(list.Files) != 1 || list.Files[0].ID != f.ID || list.NextCursor != "" || list.BaseURL != "http://example.com" {
		t.Fatalf("list status %d: %s", rec.Code, rec.Body)
	}

	req = httptest.NewRequest("DELETE", "/api/files/"+f.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if rec := do(h, req); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status %d", rec.Code)
	}
	if rec := do(h, httptest.NewRequest("GET", "/f/"+f.ID+".png", nil)); rec.Code != 404 {
		t.Fatalf("deleted file should 404, got %d", rec.Code)
	}
}

func TestAuthRequired(t *testing.T) {
	h := newTestServer(t, config.Config{})
	req := uploadReq(t, "a.png", pngBytes)
	req.Header.Set("Authorization", "Bearer wrong")
	if rec := do(h, req); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
	for _, r := range []*http.Request{
		httptest.NewRequest("GET", "/api/files", nil),
		httptest.NewRequest("DELETE", "/api/files/x", nil),
	} {
		if rec := do(h, r); rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: status %d", r.Method, r.URL, rec.Code)
		}
	}
	if rec := do(h, httptest.NewRequest("GET", "/api/health", nil)); rec.Code != 200 || !strings.Contains(rec.Body.String(), "v-test") {
		t.Fatalf("health: %d %s", rec.Code, rec.Body)
	}
}

func TestRejectsNonMedia(t *testing.T) {
	h := newTestServer(t, config.Config{})
	if rec := do(h, uploadReq(t, "page.html", []byte("<!DOCTYPE html><html></html>"))); rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("html status %d", rec.Code)
	}
	if rec := do(h, uploadReq(t, "empty.png", nil)); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty status %d", rec.Code)
	}
}

func TestRejectsOversize(t *testing.T) {
	h := newTestServer(t, config.Config{MaxSize: 1000})
	big := append(append([]byte{}, pngBytes...), bytes.Repeat([]byte{1}, 2000)...)
	if rec := do(h, uploadReq(t, "big.png", big)); rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	exact := append(append([]byte{}, pngBytes...), bytes.Repeat([]byte{1}, 1000-len(pngBytes))...)
	if rec := do(h, uploadReq(t, "exact.png", exact)); rec.Code != http.StatusCreated {
		t.Fatalf("file at the limit should pass, status %d", rec.Code)
	}
}

func TestSVGIsSandboxed(t *testing.T) {
	h := newTestServer(t, config.Config{})
	rec := do(h, uploadReq(t, "chart.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)))
	var f fileJSON
	json.Unmarshal(rec.Body.Bytes(), &f)
	rec = do(h, httptest.NewRequest("GET", "/f/"+f.ID+".svg", nil))
	if rec.Header().Get("Content-Security-Policy") != "sandbox" || rec.Header().Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("headers: %v", rec.Header())
	}
}

func TestVideoMarkdownIsLink(t *testing.T) {
	h := newTestServer(t, config.Config{})
	mp4 := append([]byte("\x00\x00\x00\x18ftypisom\x00\x00\x00\x00isommp41"), bytes.Repeat([]byte{0}, 64)...)
	rec := do(h, uploadReq(t, "demo.mp4", mp4))
	var f fileJSON
	json.Unmarshal(rec.Body.Bytes(), &f)
	if f.Type != "video/mp4" || !strings.HasPrefix(f.Markdown, "[demo.mp4](") {
		t.Fatalf("unexpected: %+v", f)
	}
}

func TestBaseURL(t *testing.T) {
	cases := []struct {
		name    string
		public  string
		host    string
		headers map[string]string
		want    string
	}{
		{"host header", "", "1.2.3.4:7000", nil, "http://1.2.3.4:7000"},
		{"forwarded", "", "127.0.0.1:8080", map[string]string{"X-Forwarded-Proto": "https", "X-Forwarded-Host": "uiv.example.com"}, "https://uiv.example.com"},
		{"forwarded list", "", "x", map[string]string{"X-Forwarded-Proto": "https, http", "X-Forwarded-Host": "a.example, b.example"}, "https://a.example"},
		{"bogus proto", "", "h.example", map[string]string{"X-Forwarded-Proto": "javascript"}, "http://h.example"},
		{"override", "https://pub.example", "h.example", map[string]string{"X-Forwarded-Host": "evil.example"}, "https://pub.example"},
	}
	for _, c := range cases {
		s := &Server{cfg: config.Config{PublicURL: c.public}}
		r := httptest.NewRequest("GET", "/", nil)
		r.Host = c.host
		for k, v := range c.headers {
			r.Header.Set(k, v)
		}
		if got := s.baseURL(r); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestStaticAndUnknownAPI(t *testing.T) {
	h := newTestServer(t, config.Config{})
	rec := do(h, httptest.NewRequest("GET", "/", nil))
	body, _ := io.ReadAll(rec.Body)
	if rec.Code != 200 || !strings.Contains(string(body), "uiv") {
		t.Fatalf("index: %d %s", rec.Code, body)
	}
	if rec := do(h, httptest.NewRequest("GET", "/api/nope", nil)); rec.Code != 404 || !strings.Contains(rec.Body.String(), "error") {
		t.Fatalf("unknown api: %d", rec.Code)
	}
}
