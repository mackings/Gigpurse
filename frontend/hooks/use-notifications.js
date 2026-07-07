"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";

export function useNotifications() {
  const { isAuthenticated } = useCurrentUser();
  const queryClient = useQueryClient();

  // Initial load only — new notifications arrive in real time over the
  // shared websocket (see RealtimeProvider), which pushes straight into this
  // query's cache. No polling.
  const query = useQuery({
    queryKey: ["notifications"],
    queryFn: () => apiGet("/notifications"),
    enabled: isAuthenticated,
  });

  const notifications = query.data || [];
  const unreadCount = notifications.filter((n) => !n.is_read).length;

  async function markAsRead(notificationId) {
    await apiPost("/notifications/read", { notification_id: notificationId });
    queryClient.invalidateQueries({ queryKey: ["notifications"] });
  }

  return { notifications, unreadCount, isLoading: query.isLoading, markAsRead };
}
