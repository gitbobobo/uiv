// Package media decides which uploads are accepted, based on file content rather than names.
package media

import (
	"bytes"
	"net/http"
	"strings"
)

// SniffLen is how many leading bytes Detect needs.
const SniffLen = 512

var extensions = map[string]string{
	"image/png":       "png",
	"image/jpeg":      "jpg",
	"image/gif":       "gif",
	"image/webp":      "webp",
	"image/bmp":       "bmp",
	"image/avif":      "avif",
	"image/heic":      "heic",
	"image/svg+xml":   "svg",
	"video/mp4":       "mp4",
	"video/webm":      "webm",
	"video/quicktime": "mov",
	"video/avi":       "avi",
}

// Detect returns the MIME type and file extension for an allowed image or video.
// ok is false when the content is not an allowed media type.
func Detect(head []byte) (mime, ext string, ok bool) {
	mime = sniff(head)
	ext, ok = extensions[mime]
	return mime, ext, ok
}

func sniff(head []byte) string {
	if brand, found := isoBrand(head); found {
		switch brand {
		case "qt  ":
			return "video/quicktime"
		case "avif", "avis":
			return "image/avif"
		case "heic", "heix", "heim", "heis", "mif1", "msf1":
			return "image/heic"
		}
	}
	mime, _, _ := strings.Cut(http.DetectContentType(head), ";")
	if mime == "text/xml" || mime == "text/plain" {
		if looksLikeSVG(head) {
			return "image/svg+xml"
		}
	}
	return mime
}

// isoBrand reads the major brand of an ISO base media file (MP4, MOV, AVIF, HEIC).
func isoBrand(head []byte) (string, bool) {
	if len(head) < 12 || string(head[4:8]) != "ftyp" {
		return "", false
	}
	return string(head[8:12]), true
}

func looksLikeSVG(head []byte) bool {
	lower := bytes.ToLower(head)
	return bytes.Contains(lower, []byte("<svg"))
}

// IsImage reports whether a stored MIME type is an image, which decides how links are rendered in Markdown.
func IsImage(mime string) bool {
	return strings.HasPrefix(mime, "image/")
}
