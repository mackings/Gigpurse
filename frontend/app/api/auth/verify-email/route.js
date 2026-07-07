import { backendFetch } from "@/lib/backend";

export async function POST(request) {
  const body = await request.json();
  const { status, envelope } = await backendFetch("/auth/email-verification/confirm", {
    method: "POST",
    body,
  });
  return Response.json(envelope, { status });
}
