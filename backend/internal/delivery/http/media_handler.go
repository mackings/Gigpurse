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

const (
	maxUploadSize     = 25 << 20 // 25MB per file
	maxFilesPerUpload = 10
)

var allowedUploadTypes = map[string]string{
	"image/jpeg":      "image",
	"image/png":       "image",
	"image/webp":      "image",
	"image/gif":       "image",
	"audio/mpeg":      "audio",
	"audio/wav":       "audio",
	"audio/ogg":       "audio",
	"audio/webm":      "audio", // what MediaRecorder produces in Chrome/Edge for voice notes
	"audio/mp4":       "audio", // what MediaRecorder produces in Safari for voice notes
	"video/mp4":       "video",
	"video/webm":      "video",
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

// uploadedFile is one item of a (possibly multi-file) upload response.
type uploadedFile struct {
	URL       string `json:"url"`
	MediaType string `json:"media_type"`
	Filename  string `json:"filename"`
}

func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize*maxFilesPerUpload)
	if err := r.ParseMultipartForm(maxUploadSize * maxFilesPerUpload); err != nil {
		respondError(w, http.StatusBadRequest, "file_too_large", "upload exceeds the size limit")
		return
	}

	// Accepts either a batch under the "files" field (a browser multipart
	// form repeats the same field name once per selected file) or a single
	// "file" field, so existing single-file callers keep working unchanged.
	headers := r.MultipartForm.File["files"]
	if len(headers) == 0 {
		if single, ok := r.MultipartForm.File["file"]; ok {
			headers = single
		}
	}
	if len(headers) == 0 {
		respondError(w, http.StatusBadRequest, "missing_file", "multipart field \"files\" (or \"file\") is required")
		return
	}
	if len(headers) > maxFilesPerUpload {
		respondError(w, http.StatusBadRequest, "too_many_files", fmt.Sprintf("at most %d files per upload", maxFilesPerUpload))
		return
	}

	if err := os.MkdirAll(h.uploadDir, 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "upload_dir_failed", "failed to prepare upload storage")
		return
	}

	uploaded := make([]uploadedFile, 0, len(headers))
	for _, header := range headers {
		if header.Size > maxUploadSize {
			respondError(w, http.StatusBadRequest, "file_too_large", fmt.Sprintf("%q exceeds the 25MB upload limit", header.Filename))
			return
		}

		contentType := header.Header.Get("Content-Type")
		// MediaRecorder sends codec parameters along with the type (e.g.
		// "audio/webm;codecs=opus") — strip everything after the ";" before
		// checking the allowlist, which only keys on the base type.
		baseContentType := strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])
		mediaType, ok := allowedUploadTypes[baseContentType]
		if !ok {
			respondError(w, http.StatusBadRequest, "unsupported_media_type", fmt.Sprintf("unsupported content type for %q: %s", header.Filename, contentType))
			return
		}

		file, err := header.Open()
		if err != nil {
			respondError(w, http.StatusBadRequest, "upload_failed", fmt.Sprintf("failed to read %q", header.Filename))
			return
		}

		name, err := randomFilename(filepath.Ext(header.Filename))
		if err != nil {
			file.Close()
			respondError(w, http.StatusInternalServerError, "upload_failed", "failed to generate file name")
			return
		}

		dstPath := filepath.Join(h.uploadDir, name)
		dst, err := os.Create(dstPath)
		if err != nil {
			file.Close()
			respondError(w, http.StatusInternalServerError, "upload_failed", "failed to store file")
			return
		}

		_, copyErr := io.Copy(dst, file)
		file.Close()
		dst.Close()
		if copyErr != nil {
			respondError(w, http.StatusInternalServerError, "upload_failed", "failed to store file")
			return
		}

		uploaded = append(uploaded, uploadedFile{
			URL:       h.publicURL + "/uploads/" + name,
			MediaType: mediaType,
			Filename:  header.Filename,
		})
	}

	// Single-file requests keep the original flat response shape so
	// existing callers don't need to change; batch requests get "files".
	if len(uploaded) == 1 {
		respondSuccess(w, http.StatusCreated, "file uploaded successfully", map[string]any{
			"url":        uploaded[0].URL,
			"media_type": uploaded[0].MediaType,
			"files":      uploaded,
		})
		return
	}
	respondSuccess(w, http.StatusCreated, "files uploaded successfully", map[string]any{"files": uploaded})
}

func randomFilename(ext string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ext, nil
}
