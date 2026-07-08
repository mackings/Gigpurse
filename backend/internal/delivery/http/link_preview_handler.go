package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// LinkPreviewHandler unfurls a URL into a title/thumbnail/embed so the
// portfolio editor can show a rich card instead of a bare link — known
// platforms go through their oEmbed API, everything else falls back to
// scraping Open Graph tags.
type LinkPreviewHandler struct {
	client *http.Client
}

func NewLinkPreviewHandler() *LinkPreviewHandler {
	return &LinkPreviewHandler{client: &http.Client{Timeout: 8 * time.Second}}
}

func (h *LinkPreviewHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/link-preview", JWTMiddleware(h.Preview))
}

type linkPreview struct {
	URL          string `json:"url"`
	Title        string `json:"title"`
	Description  string `json:"description,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	EmbedURL     string `json:"embed_url,omitempty"`
	Provider     string `json:"provider"`
	MediaType    string `json:"media_type"` // "video", "audio", or "link"
}

func (h *LinkPreviewHandler) Preview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	raw := r.URL.Query().Get("url")
	if raw == "" {
		respondError(w, http.StatusBadRequest, "missing_url", "url query parameter is required")
		return
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		respondError(w, http.StatusBadRequest, "invalid_url", "url must be a valid http(s) URL")
		return
	}
	if isDisallowedHost(parsed.Hostname()) {
		respondError(w, http.StatusBadRequest, "invalid_url", "that URL's host isn't allowed")
		return
	}

	preview, err := h.fetch(parsed)
	if err != nil {
		respondError(w, http.StatusBadGateway, "preview_failed", "couldn't generate a preview for that link")
		return
	}
	respondSuccess(w, http.StatusOK, "link preview retrieved successfully", preview)
}

func (h *LinkPreviewHandler) fetch(u *url.URL) (*linkPreview, error) {
	host := strings.ToLower(u.Hostname())
	original := u.String()
	switch {
	case strings.Contains(host, "youtube.com") || host == "youtu.be":
		return h.oembed(original, "https://www.youtube.com/oembed?format=json&url=", "youtube", "video")
	case strings.Contains(host, "vimeo.com"):
		return h.oembed(original, "https://vimeo.com/api/oembed.json?url=", "vimeo", "video")
	case strings.Contains(host, "soundcloud.com"):
		return h.oembed(original, "https://soundcloud.com/oembed?format=json&url=", "soundcloud", "audio")
	case strings.Contains(host, "open.spotify.com"):
		return h.oembed(original, "https://open.spotify.com/oembed?url=", "spotify", "audio")
	case strings.Contains(host, "tiktok.com"):
		return h.oembed(original, "https://www.tiktok.com/oembed?url=", "tiktok", "video")
	default:
		return h.openGraph(u)
	}
}

type oembedResponse struct {
	Title        string `json:"title"`
	ThumbnailURL string `json:"thumbnail_url"`
	HTML         string `json:"html"`
	AuthorName   string `json:"author_name"`
	ProviderName string `json:"provider_name"`
}

var iframeSrcRe = regexp.MustCompile(`src="([^"]+)"`)

func (h *LinkPreviewHandler) oembed(originalURL, endpoint, provider, mediaType string) (*linkPreview, error) {
	resp, err := h.client.Get(endpoint + url.QueryEscape(originalURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oembed %s responded with status %d", provider, resp.StatusCode)
	}

	var data oembedResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		return nil, err
	}

	title := data.Title
	if title == "" {
		title = data.AuthorName
	}
	var embedURL string
	if m := iframeSrcRe.FindStringSubmatch(data.HTML); len(m) == 2 {
		embedURL = html.UnescapeString(m[1])
	}

	return &linkPreview{
		URL:          originalURL,
		Title:        title,
		ThumbnailURL: data.ThumbnailURL,
		EmbedURL:     embedURL,
		Provider:     provider,
		MediaType:    mediaType,
	}, nil
}

// openGraph is the fallback for any URL that isn't a known media platform —
// most sites (portfolios, articles, press mentions, Behance/Dribbble
// projects, etc.) publish og:title/og:image, so this covers "paste any
// link" generally rather than only a fixed allowlist of platforms.
func (h *LinkPreviewHandler) openGraph(u *url.URL) (*linkPreview, error) {
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GigPurseLinkPreview/1.0; +https://gigpurse.app)")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch responded with status %d", resp.StatusCode)
	}

	doc, err := html.Parse(io.LimitReader(resp.Body, 2<<20)) // 2MB cap
	if err != nil {
		return nil, err
	}

	preview := &linkPreview{URL: u.String(), Provider: "link", MediaType: "link"}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "meta":
				var key, content string
				for _, a := range n.Attr {
					switch a.Key {
					case "property", "name":
						if key == "" {
							key = a.Val
						}
					case "content":
						content = a.Val
					}
				}
				switch key {
				case "og:title":
					preview.Title = content
				case "og:description", "description":
					if preview.Description == "" {
						preview.Description = content
					}
				case "og:image":
					preview.ThumbnailURL = content
				}
			case "title":
				if preview.Title == "" && n.FirstChild != nil {
					preview.Title = strings.TrimSpace(n.FirstChild.Data)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	if preview.Title == "" {
		preview.Title = u.Hostname()
	}
	return preview, nil
}

// isDisallowedHost is a basic SSRF guard: this endpoint fetches whatever
// URL a caller supplies, so it must not be usable to probe internal/private
// network addresses from the server's vantage point.
func isDisallowedHost(host string) bool {
	if host == "" || host == "localhost" {
		return true
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return true // fail closed — can't verify it's safe, so don't fetch it
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return true
		}
	}
	return false
}
