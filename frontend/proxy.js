import { NextResponse } from "next/server";
import jwt from "jsonwebtoken";

const COOKIE_NAME = "gigpurse_token";

function getSessionPayload(request) {
  const token = request.cookies.get(COOKIE_NAME)?.value;
  if (!token) return null;
  try {
    const payload = jwt.decode(token);
    if (!payload?.exp || payload.exp * 1000 <= Date.now()) return null;
    return payload;
  } catch {
    return null;
  }
}

export function proxy(request) {
  const session = getSessionPayload(request);
  if (!session) {
    const loginUrl = new URL("/login", request.url);
    loginUrl.searchParams.set("next", request.nextUrl.pathname);
    return NextResponse.redirect(loginUrl);
  }

  if (request.nextUrl.pathname.startsWith("/admin")) {
    const isDisputesArea = request.nextUrl.pathname.startsWith("/admin/disputes");
    const allowed = session.role === "admin" || (session.role === "moderator" && isDisputesArea);
    if (!allowed) {
      return NextResponse.redirect(new URL("/", request.url));
    }
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    "/dashboard/:path*",
    "/onboarding",
    "/portfolio",
    "/jobs/post",
    "/jobs/mine",
    "/messages",
    "/wallet",
    "/disputes",
    "/contracts/:path*",
    "/profile",
    "/admin/:path*",
  ],
};
