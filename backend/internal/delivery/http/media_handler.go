package http

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const maxUploadSize = 25 << 20 // 25MB

var allowedUploadTypes = map[string]string{
	"image/jpeg": "image",
	"image/png":  "image",
	"image/webp": "image",
	"image/gif":  "image",
	"audio/mpeg": "audio",
	"audio/wav":  "audio",
	"audio/ogg":  "audio",
	"video/mp4":  "video",
	"video/webm": "video",
	"video/quicktime": "video",
}

type MediaHandler struct {
	uploadDir string
	publicURL string
}

// NewMediaHandler stores uploads on local disk under uploadDir and serves them
// back from publicBaseURL + "/uploads/...". This is intentionally simple
// (no domain/usecase indirection) — there's no business rule here, just file
// I/O. Swapping to cloud storage later is a self-contained change to this
// one file, not a cross-cutting refactor.
func NewMediaHandler(uploadDir, publicBaseURL string) *MediaHandler {
	return &MediaHandler{uploadDir: uploadDir, publicURL: strings.TrimRight(publicBaseURL, "/")}
}

func (h *MediaHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/media/upload", JWTMiddleware(h.Upload))
}

func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		respondError(w, http.StatusBadRequest, "file_too_large", "file exceeds the 25MB upload limit")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "missing_file", "multipart field \"file\" is required")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	mediaType, ok := allowedUploadTypes[contentType]
	if !ok {
		respondError(w, http.StatusBadRequest, "unsupported_media_type", fmt.Sprintf("unsupported content type: %s", contentType))
		return
	}

	if err := os.MkdirAll(h.uploadDir, 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "upload_dir_failed", "failed to prepare upload storage")
		return
	}

	name, err := randomFilename(filepath.Ext(header.Filename))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "upload_failed", "failed to generate file name")
		return
	}

	dstPath := filepath.Join(h.uploadDir, name)
	dst, err := os.Create(dstPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "upload_failed", "failed to store file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		respondError(w, http.StatusInternalServerError, "upload_failed", "failed to store file")
		return
	}

	respondSuccess(w, http.StatusCreated, "file uploaded successfully", map[string]string{
		"url":        h.publicURL + "/uploads/" + name,
		"media_type": mediaType,
	})
}

func randomFilename(ext string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ext, nil
}
