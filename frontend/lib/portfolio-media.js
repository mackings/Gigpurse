// Shared semantics for a portfolio item, used by both the editor and the
// public profile grid:
//   - Uploaded file (image/video/audio): media_type set, url points at the
//     file itself, no external_url.
//   - Link-based item (YouTube, Vimeo, SoundCloud, Spotify, TikTok, or any
//     other site): url is the original page the talent pasted, external_url
//     is the embeddable player URL (only set when the provider is
//     embeddable), thumbnail_url is the preview image. media_type is
//     "video", "audio", or "link".
export function isLinkItem(item) {
  return !!item.external_url || item.media_type === "link";
}

export function isEmbeddable(item) {
  return isLinkItem(item) && !!item.external_url;
}

export function domainOf(url) {
  try {
    return new URL(url).hostname.replace(/^www\./, "");
  } catch {
    return url;
  }
}
