import { backendFetch } from "@/lib/backend";
import { setSessionCookie } from "@/lib/session";

export async function POST(request) {
  const body = await request.json();
  const { status, envelope } = await backendFetch("/auth/moderator/verify", {
    method: "POST",
    body,
  });

  if (!envelope?.success || !envelope.data?.token) {
    return Response.json(envelope, { status });
  }

  await setSessionCookie(envelope.data.token);

  return Response.json(
    { ...envelope, data: { user: envelope.data.user } },
    { status }
  );
}
