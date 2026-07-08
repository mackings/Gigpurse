import { NextResponse } from "next/server";

// Cheap, unauthenticated endpoint for the backend's keep-alive ping — no
// rendering, no backend calls, just proves the frontend process is up.
export async function GET() {
  return NextResponse.json({ status: "ok" });
}
