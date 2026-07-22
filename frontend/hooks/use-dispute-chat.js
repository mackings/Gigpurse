"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "@/lib/api";
import { useRealtime } from "@/lib/RealtimeProvider";

// Mirrors useMilestones' shape — a single hook owning both the dispute
// record and its chat thread, backed by the shared realtime socket for
// sending (REST is only ever a fallback the send functions don't use
// directly; the socket queues while reconnecting, same as normal chat).
export function useDisputeChat(disputeId) {
  const queryClient = useQueryClient();
  const { sendDisputeMessage: sendOverSocket } = useRealtime();

  const disputeQuery = useQuery({
    queryKey: ["dispute", disputeId],
    queryFn: () => apiGet(`/disputes?id=${disputeId}`),
    enabled: !!disputeId,
  });

  const messagesQuery = useQuery({
    queryKey: ["dispute-messages", disputeId],
    queryFn: () => apiGet(`/disputes/messages?dispute_id=${disputeId}`),
    enabled: !!disputeId,
  });

  function invalidate() {
    queryClient.invalidateQueries({ queryKey: ["dispute", disputeId] });
    queryClient.invalidateQueries({ queryKey: ["disputes"] });
  }

  function sendMessage(content, opts = {}) {
    return sendOverSocket(disputeId, content, opts);
  }

  async function join() {
    const dispute = await apiPost("/disputes/join", { dispute_id: disputeId });
    invalidate();
    return dispute;
  }

  async function resolve(winnerId, resolution) {
    const dispute = await apiPost("/admin/disputes/resolve", { dispute_id: disputeId, winner_id: winnerId, resolution });
    invalidate();
    return dispute;
  }

  return {
    dispute: disputeQuery.data,
    isLoadingDispute: disputeQuery.isLoading,
    messages: messagesQuery.data || [],
    isLoadingMessages: messagesQuery.isLoading,
    sendMessage,
    join,
    resolve,
  };
}
