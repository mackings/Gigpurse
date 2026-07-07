import { clearSessionCookie } from "@/lib/session";

export async function POST() {
  await clearSessionCookie();
  return Response.json({ success: true, status: "success", status_code: 200, message: "logged out" });
}
