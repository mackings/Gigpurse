import { getSessionToken, decodeToken } from "@/lib/session";

// Narrow exception to "the client never sees the JWT": the chat WebSocket
// connects directly to the Go backend and Next.js Route Handlers can't proxy
// an upgraded WebSocket connection, so the client needs the raw token once.
export async function GET() {
  const token = await getSessionToken();
  if (!token || !decodeToken(token)) {
    return Response.json({ success: false, message: "not authenticated" }, { status: 401 });
  }
  return Response.json({ success: true, data: { token } });
}
