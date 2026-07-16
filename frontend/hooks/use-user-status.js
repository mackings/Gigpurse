"use client";

import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";

// Presence needs to feel closer to live than a name/bio does, so this polls
// on its own short interval rather than reusing useUserInfo's long,
// rarely-refetched cache — there's no websocket presence-broadcast, so
// short polling is the simplest way to keep the indicator reasonably fresh.
export function useUserStatus(userId) {
  const query = useQuery({
    queryKey: ["user-status", userId],
    queryFn: () => apiGet(`/users/${userId}`),
    enabled: !!userId,
    staleTime: 15 * 1000,
    refetchInterval: 20 * 1000,
  });
  return query.data?.status; // "online" | "offline" | "disabled" | undefined
}
