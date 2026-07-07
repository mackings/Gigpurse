import { NextResponse } from "next/server";
import { getSessionToken } from "@/lib/session";
import { API_URL } from "@/lib/backend";

async function handle(request, { params }, method) {
  const { path } = await params;
  const token = await getSessionToken();

  const url = `${API_URL}/${path.join("/")}${request.nextUrl.search}`;
  const headers = {};
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const hasBody = method === "POST" || method === "PUT" || method === "DELETE";
  let body;
  const incomingContentType = request.headers.get("content-type") || "";
  if (hasBody && incomingContentType.startsWith("multipart/form-data")) {
    headers["Content-Type"] = incomingContentType;
    body = await request.arrayBuffer();
  } else if (hasBody) {
    headers["Content-Type"] = "application/json";
    const raw = await request.text();
    body = raw.length > 0 ? raw : undefined;
  }

  const res = await fetch(url, { method, headers, body, cache: "no-store" });
  const envelope = await res.json().catch(() => null);
  return NextResponse.json(envelope, { status: res.status });
}

export async function GET(request, ctx) {
  return handle(request, ctx, "GET");
}
export async function POST(request, ctx) {
  return handle(request, ctx, "POST");
}
export async function PUT(request, ctx) {
  return handle(request, ctx, "PUT");
}
export async function DELETE(request, ctx) {
  return handle(request, ctx, "DELETE");
}
