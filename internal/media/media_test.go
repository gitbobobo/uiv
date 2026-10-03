package media

import "testing"

func ftyp(brand string) []byte {
	b := []byte{0, 0, 0, 0x18}
	b = append(b, "ftyp"...)
	b = append(b, brand...)
	b = append(b, 0, 0, 0, 0)
	b = append(b, "isommp41"...)
	return b
}

func TestDetect(t *testing.T) {
	cases := []struct {
		name string
		head []byte
		mime string
		ext  string
		ok   bool
	}{
		{"png", []byte("\x89PNG\r\n\x1a\n0000"), "image/png", "png", true},
		{"jpeg", []byte("\xff\xd8\xff\xe0000000"), "image/jpeg", "jpg", true},
		{"gif", []byte("GIF89a000000"), "image/gif", "gif", true},
		{"webp", []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), "image/webp", "webp", true},
		{"mp4", ftyp("isom"), "video/mp4", "mp4", true},
		{"mov", ftyp("qt  "), "video/quicktime", "mov", true},
		{"avif", ftyp("avif"), "image/avif", "avif", true},
		{"heic", ftyp("heic"), "image/heic", "heic", true},
		{"webm", []byte("\x1a\x45\xdf\xa3\x01\x00\x00\x00"), "video/webm", "webm", true},
		{"svg", []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"></svg>`), "image/svg+xml", "svg", true},
		{"bare svg", []byte(`<svg viewBox="0 0 1 1"></svg>`), "image/svg+xml", "svg", true},
		{"html", []byte(`<!DOCTYPE html><html><svg></svg></html>`), "text/html", "", false},
		{"text", []byte("hello world"), "text/plain", "", false},
		{"zip", []byte("PK\x03\x04000000"), "application/zip", "", false},
	}
	for _, c := range cases {
		mime, ext, ok := Detect(c.head)
		if mime != c.mime || ext != c.ext || ok != c.ok {
			t.Errorf("%s: got (%q, %q, %v), want (%q, %q, %v)", c.name, mime, ext, ok, c.mime, c.ext, c.ok)
		}
	}
}
