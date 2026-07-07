import "server-only";
import { cookies } from "next/headers";
import jwt from "jsonwebtoken";

const COOKIE_NAME = "gigpurse_token";
const MAX_AGE_SECONDS = 60 * 60 * 24; // 24h, matches the backend JWT expiry

export async function setSessionCookie(token) {
  const cookieStore = await cookies();
  cookieStore.set(COOKIE_NAME, token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: MAX_AGE_SECONDS,
  });
}

export async function clearSessionCookie() {
  const cookieStore = await cookies();
  cookieStore.delete(COOKIE_NAME);
}

export async function getSessionToken() {
  const cookieStore = await cookies();
  return cookieStore.get(COOKIE_NAME)?.value ?? null;
}

// Optimistic decode only (no signature verification) — the Go backend is the
// actual authorization boundary and re-validates the JWT on every proxied request.
export function decodeToken(token) {
  if (!token) return null;
  try {
    const payload = jwt.decode(token);
    if (!payload || typeof payload.exp !== "number") return null;
    if (payload.exp * 1000 < Date.now()) return null;
    return payload;
  } catch {
    return null;
  }
}

export async function getSession() {
  const token = await getSessionToken();
  return decodeToken(token);
}

export { COOKIE_NAME };
