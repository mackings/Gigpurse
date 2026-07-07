import { getWsToken } from "@/lib/api";

// The realtime socket connects directly to the Go backend — Next.js Route
// Handlers can't proxy an upgraded WebSocket connection — so this is the one
// place the client needs a public backend URL (not the server-only proxy base).
const PUBLIC_API_URL = process.env.NEXT_PUBLIC_GIGPURSE_API_URL || "http://localhost:8080";

// One socket per session carries both chat messages and notification pushes
// (see RealtimeProvider). Opened once app-wide, not per chat window.
export async function openRealtimeSocket() {
  const token = await getWsToken();
  if (!token) return null;
  const wsUrl = PUBLIC_API_URL.replace(/^http/, "ws") + `/ws?token=${encodeURIComponent(token)}`;
  return new WebSocket(wsUrl);
}
