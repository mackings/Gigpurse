"use client";

import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";

export function useCurrentUser() {
  const query = useQuery({
    queryKey: ["profile", "me"],
    queryFn: () => apiGet("/users/profile"),
    retry: false,
  });

  return {
    user: query.data ?? null,
    isLoading: query.isLoading,
    isAuthenticated: !!query.data && !query.isError,
    refetch: query.refetch,
  };
}
