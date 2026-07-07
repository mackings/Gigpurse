"use client";

import { useRef, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Bell, CheckCheck, BellRing, BellOff, Loader2 } from "lucide-react";
import { useNotifications } from "@/hooks/use-notifications";
import { usePushNotifications } from "@/hooks/use-push-notifications";
import { toast } from "sonner";

function timeAgo(dateStr) {
  const diffMs = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diffMs / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

export default function NotificationBell({ className }) {
  const { notifications, unreadCount, markAsRead } = useNotifications();
  const { supported: pushSupported, subscribed: pushSubscribed, isLoading: pushLoading, subscribe: pushSubscribe, unsubscribe: pushUnsubscribe } = usePushNotifications();
  const router = useRouter();
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  // Radix's MenuItem closes the whole dropdown synchronously, in the same
  // click handler, as part of selecting an item — which unmounts the menu
  // (and anything inside it, link or not) before a browser gets to process
  // that click's default navigate action, silently cancelling it. The fix
  // is to not navigate from the item's own handler at all: record what to
  // do, and only actually navigate from onOpenChange once Radix confirms
  // the menu has fully closed.
  const pendingNavRef = useRef(null);
  // Separately: re-navigating to a URL you're already on (same path, same
  // hash) is a Next.js router no-op — and pushing the same hash again can
  // even produce a malformed doubled-hash URL. For "already here" cases
  // with a target section, scroll straight to it instead of routing.
  const pendingScrollRef = useRef(null);

  function selectNotification(n) {
    if (!n.is_read) markAsRead(n.id);
    const target = n.link || (n.contract_id ? `/messages?contract=${n.contract_id}` : null);
    if (!target) return;

    const [targetPathAndQuery, targetHash] = target.split("#");
    const targetPath = targetPathAndQuery.split("?")[0];

    if (targetPath === pathname) {
      pendingNavRef.current = null;
      if (targetHash) {
        pendingScrollRef.current = targetHash;
      } else {
        toast.success("You're already on this page — it's up to date.");
      }
      return;
    }
    pendingScrollRef.current = null;
    pendingNavRef.current = target;
  }

  async function togglePush(e) {
    e.preventDefault();
    try {
      if (pushSubscribed) {
        await pushUnsubscribe();
        toast.success("Push notifications turned off for this device.");
      } else {
        await pushSubscribe();
        toast.success("Push notifications enabled for this device.");
      }
    } catch (err) {
      toast.error(err.message);
    }
  }

  function handleOpenChange(nextOpen) {
    setOpen(nextOpen);
    if (nextOpen) return;
    if (pendingNavRef.current) {
      const target = pendingNavRef.current;
      pendingNavRef.current = null;
      router.push(target);
    } else if (pendingScrollRef.current) {
      const hash = pendingScrollRef.current;
      pendingScrollRef.current = null;
      document.getElementById(hash)?.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  }

  return (
    <DropdownMenu open={open} onOpenChange={handleOpenChange}>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className={`relative ${className || ""}`}>
          <Bell className="w-4 h-4" />
          {unreadCount > 0 && (
            <span className="absolute -top-0.5 -right-0.5 w-4 h-4 rounded-full bg-primary text-primary-foreground text-[10px] font-semibold flex items-center justify-center">
              {unreadCount > 9 ? "9+" : unreadCount}
            </span>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-80 max-h-96 overflow-y-auto">
        <div className="px-2 py-1.5 flex items-center justify-between">
          <p className="text-sm font-medium">Notifications</p>
          {unreadCount > 0 && <span className="text-xs text-muted-foreground">{unreadCount} unread</span>}
        </div>
        {pushSupported && (
          <DropdownMenuItem
            onSelect={(e) => e.preventDefault()}
            onClick={togglePush}
            disabled={pushLoading}
            className="gap-2 text-xs text-muted-foreground justify-center mb-1"
          >
            {pushLoading ? (
              <Loader2 className="w-3.5 h-3.5 animate-spin" />
            ) : pushSubscribed ? (
              <BellOff className="w-3.5 h-3.5" />
            ) : (
              <BellRing className="w-3.5 h-3.5" />
            )}
            {pushSubscribed ? "Turn off push notifications" : "Enable push notifications"}
          </DropdownMenuItem>
        )}
        {notifications.length === 0 ? (
          <p className="px-2 py-6 text-sm text-muted-foreground text-center">You&apos;re all caught up.</p>
        ) : (
          notifications.map((n) => (
            <DropdownMenuItem
              key={n.id}
              onSelect={() => selectNotification(n)}
              className={`flex flex-col items-start gap-0.5 whitespace-normal ${!n.is_read ? "bg-accent/60" : ""}`}
            >
              <div className="flex items-center gap-1.5 w-full">
                {!n.is_read && <span className="w-1.5 h-1.5 rounded-full bg-primary shrink-0" />}
                <p className="text-sm font-medium truncate">{n.title}</p>
              </div>
              <p className="text-xs text-muted-foreground">{n.message}</p>
              <p className="text-[11px] text-muted-foreground/70">{timeAgo(n.created_at)}</p>
            </DropdownMenuItem>
          ))
        )}
        {unreadCount > 0 && (
          <DropdownMenuItem
            onSelect={() => notifications.filter((n) => !n.is_read).forEach((n) => markAsRead(n.id))}
            className="gap-2 text-xs text-muted-foreground justify-center mt-1"
          >
            <CheckCheck className="w-3.5 h-3.5" />
            Mark all as read
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
