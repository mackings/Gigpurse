"use client";

import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";

// Resolves any user's display name/role by ID (client or musician — chat
// partners can be either). Names change rarely, so this is cached long and
// never refetched aggressively.
export function useUserInfo(userId) {
  const query = useQuery({
    queryKey: ["user-info", userId],
    queryFn: () => apiGet(`/users/${userId}`),
    enabled: !!userId,
    staleTime: 5 * 60 * 1000,
  });
  return query.data;
}
