async function request(path, { method = "GET", body } = {}) {
  const res = await fetch(`/api/proxy${path}`, {
    method,
    headers: body !== undefined ? { "Content-Type": "application/json" } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
    credentials: "include",
  });
  const envelope = await res.json().catch(() => null);
  if (!res.ok || !envelope?.success) {
    const message = envelope?.message || envelope?.error?.message || "Request failed";
    const error = new Error(message);
    error.status = res.status;
    error.code = envelope?.error?.code;
    throw error;
  }
  return envelope.data;
}

export const apiGet = (path) => request(path);
export const apiPost = (path, body) => request(path, { method: "POST", body });
export const apiPut = (path, body) => request(path, { method: "PUT", body });
export const apiDelete = (path, body) => request(path, { method: "DELETE", body });

// Accepts a single File or an array of Files — either way, returns
// { url, media_type, files: [...] } (single) or { files: [...] } (batch).
export async function apiUpload(path, files) {
  const formData = new FormData();
  const list = Array.isArray(files) ? files : [files];
  list.forEach((file) => formData.append("files", file));
  const res = await fetch(`/api/proxy${path}`, {
    method: "POST",
    body: formData,
    credentials: "include",
  });
  const envelope = await res.json().catch(() => null);
  if (!res.ok || !envelope?.success) {
    const message = envelope?.message || envelope?.error?.message || "Upload failed";
    const error = new Error(message);
    error.status = res.status;
    throw error;
  }
  return envelope.data;
}

async function authRequest(path, body) {
  const res = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    credentials: "include",
  });
  const envelope = await res.json().catch(() => null);
  if (!res.ok || !envelope?.success) {
    const message = envelope?.message || envelope?.error?.message || "Request failed";
    const error = new Error(message);
    error.status = res.status;
    throw error;
  }
  return envelope.data;
}

export const signup = (payload) => authRequest("/api/auth/signup", payload);
export const login = (payload) => authRequest("/api/auth/login", payload);
export const logout = () => authRequest("/api/auth/logout", {});
export const resendVerification = (email) => authRequest("/api/auth/resend-verification", { email });
export const verifyEmail = (payload) => authRequest("/api/auth/verify-email", payload);
export const forgotPassword = (email) => authRequest("/api/auth/forgot-password", { email });
export const resetPassword = (payload) => authRequest("/api/auth/reset-password", payload);

export async function getWsToken() {
  const res = await fetch("/api/auth/ws-token", { credentials: "include" });
  const envelope = await res.json().catch(() => null);
  if (!res.ok || !envelope?.success) return null;
  return envelope.data.token;
}
