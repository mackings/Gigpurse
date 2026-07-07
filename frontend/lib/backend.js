import "server-only";

const API_URL = process.env.GIGPURSE_API_URL || "http://localhost:8080";

// Calls the Go backend directly. Used by the auth route handlers, which need
// to inspect/set the session cookie themselves rather than blindly relaying.
export async function backendFetch(path, { method = "GET", token, body } = {}) {
  const headers = { "Content-Type": "application/json" };
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(`${API_URL}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
    cache: "no-store",
  });

  const envelope = await res.json().catch(() => null);
  return { status: res.status, envelope };
}

export { API_URL };
