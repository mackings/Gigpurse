"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "@/lib/api";

// All of the current user's direct-hire (booking) requests — used to
// resolve "is there a pending booking with this chat partner" when a chat
// is opened without an explicit ?booking= id.
export function useDirectHires() {
  const query = useQuery({
    queryKey: ["direct-hires", "all-mine"],
    queryFn: () => apiGet("/direct-hires"),
  });
  return { requests: query.data || [], isLoading: query.isLoading };
}

export function useDirectHire(requestId) {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ["direct-hire", requestId],
    queryFn: () => apiGet(`/direct-hires?id=${requestId}`),
    enabled: !!requestId,
  });

  function invalidate() {
    queryClient.invalidateQueries({ queryKey: ["direct-hire", requestId] });
    queryClient.invalidateQueries({ queryKey: ["direct-hires"] });
  }

  async function accept() {
    const r = await apiPost("/direct-hires/respond", { request_id: requestId, decision: "accepted" });
    invalidate();
    return r;
  }

  async function decline() {
    const r = await apiPost("/direct-hires/respond", { request_id: requestId, decision: "declined" });
    invalidate();
    return r;
  }

  async function counter(terms) {
    const r = await apiPost("/direct-hires/counter", { request_id: requestId, ...terms });
    invalidate();
    return r;
  }

  return { request: query.data, isLoading: query.isLoading, accept, decline, counter };
}
