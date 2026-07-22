"use client";

import { createContext, useContext, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { openRealtimeSocket } from "@/lib/ws";
import { useCurrentUser } from "@/hooks/use-current-user";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogAction,
} from "@/components/ui/alert-dialog";

const RealtimeContext = createContext({
  connected: false,
  sendChatMessage: () => false,
  sendDisputeMessage: () => false,
  pendingSendCount: 0,
  unreadMessageCount: 0,
  unreadByPartner: {},
  unreadByDispute: {},
  clearUnreadMessages: () => {},
  clearUnreadForPartner: () => {},
  clearUnreadForDispute: () => {},
});

export function useRealtime() {
  return useContext(RealtimeContext);
}

function playNotificationSound() {
  try {
    const audio = new Audio("/notification.wav");
    audio.volume = 0.5;
    // Autoplay can be blocked before the user has interacted with the page
    // at all; that's fine, this just becomes a silent no-op in that case.
    audio.play().catch(() => {});
  } catch {
    // Audio unsupported in this environment — non-fatal.
  }
}

// Single app-wide websocket, opened once per authenticated session. Every
// chat message and notification arrives here and is pushed straight into
// the relevant React Query caches — no component owns its own connection
// and nothing polls. ChatWindow/ChatList/NotificationBell just read the
// query caches this keeps up to date.
export function RealtimeProvider({ children }) {
  const { user, isAuthenticated } = useCurrentUser();
  const queryClient = useQueryClient();
  const router = useRouter();
  const [connected, setConnected] = useState(false);
  const [unreadByPartner, setUnreadByPartner] = useState({});
  const [unreadByDispute, setUnreadByDispute] = useState({});
  const [pendingSendCount, setPendingSendCount] = useState(0);
  // The websocket send path (sendChatMessage below) is fire-and-forget —
  // there's no request/response round trip to hang a try/catch off. The
  // server pushes a `type: "error"` frame back on rejection (e.g. the
  // recipient disabled their account) instead, surfaced here as a blocking
  // dialog rather than a toast, since a silently-dropped message deserves
  // more than something that can be missed.
  const [sendError, setSendError] = useState(null);
  const socketRef = useRef(null);
  const userIdRef = useRef(user?.id);
  // Outgoing sends attempted while the socket is down (e.g. the brief
  // "Connecting..." window on load or after a network blip) queue here
  // instead of being silently dropped, and flush the instant the socket
  // reopens — this is what actually fixes messages disappearing.
  const pendingQueueRef = useRef([]);

  useEffect(() => {
    userIdRef.current = user?.id;
  }, [user?.id]);

  useEffect(() => {
    if (!isAuthenticated) return undefined;
    let stopped = false;
    let reconnectTimer;

    async function connect() {
      const socket = await openRealtimeSocket();
      if (!socket || stopped) return;
      socketRef.current = socket;

      socket.onopen = () => {
        setConnected(true);
        while (pendingQueueRef.current.length) {
          socket.send(pendingQueueRef.current.shift());
        }
        setPendingSendCount(0);
      };

      socket.onclose = () => {
        setConnected(false);
        socketRef.current = null;
        if (stopped) return;
        reconnectTimer = setTimeout(connect, 2000);
      };

      socket.onmessage = (event) => {
        let envelope;
        try {
          envelope = JSON.parse(event.data);
        } catch {
          return;
        }

        if (envelope.type === "chat_message") {
          const msg = envelope.data;
          const myId = userIdRef.current;
          const partnerId = msg.sender_id === myId ? msg.recv_id : msg.sender_id;
          queryClient.setQueryData(["chat-history", partnerId], (old) => {
            const list = old || [];
            if (list.some((m) => m.id === msg.id)) return list;
            return [...list, msg];
          });
          queryClient.invalidateQueries({ queryKey: ["chats-recent"] });
          if (msg.recv_id === myId) {
            setUnreadByPartner((prev) => ({ ...prev, [msg.sender_id]: (prev[msg.sender_id] || 0) + 1 }));
          }
        } else if (envelope.type === "dispute_message") {
          const msg = envelope.data;
          const myId = userIdRef.current;
          queryClient.setQueryData(["dispute-messages", msg.dispute_id], (old) => {
            const list = old || [];
            if (list.some((m) => m.id === msg.id)) return list;
            return [...list, msg];
          });
          if (msg.sender_id !== myId) {
            setUnreadByDispute((prev) => ({ ...prev, [msg.dispute_id]: (prev[msg.dispute_id] || 0) + 1 }));
          }
        } else if (envelope.type === "dispute_updated") {
          const dispute = envelope.data;
          queryClient.setQueryData(["dispute", dispute.id], dispute);
          queryClient.invalidateQueries({ queryKey: ["disputes"] });
          // Status changes (moderator joining, dispute resolving) also drop a
          // system message into the room — refetch so it shows up live
          // instead of only after the chat is reopened.
          queryClient.invalidateQueries({ queryKey: ["dispute-messages", dispute.id] });
        } else if (envelope.type === "notification") {
          const notif = envelope.data;
          queryClient.setQueryData(["notifications"], (old) => {
            const list = old || [];
            if (list.some((n) => n.id === notif.id)) return list;
            return [notif, ...list];
          });
          playNotificationSound();
          // Visible pop-up on top of the sound+badge+cache update above —
          // without this, a live notification only showed up silently in
          // the bell dropdown, easy to miss if you weren't looking at it.
          toast(notif.title, {
            description: notif.message,
            action: notif.link
              ? { label: "View", onClick: () => router.push(notif.link) }
              : undefined,
          });
          // Milestone/escrow events carry contract_id — refresh that
          // contract's milestones and the wallet live, so a client funding
          // escrow (or releasing a payment) shows up on the musician's
          // screen instantly without them navigating anywhere.
          if (notif.contract_id) {
            queryClient.invalidateQueries({ queryKey: ["milestones", notif.contract_id] });
          }
          // Booking (direct-hire) notifications carry a link like
          // /messages?with=...&booking=<id> — pull the id out so the
          // specific booking's own cache entry (["direct-hire", id], used
          // by the chat-embedded negotiation panel) gets refreshed too, not
          // just the list view. These are two different query keys
          // ("direct-hire" singular vs "direct-hires" plural) and
          // invalidating one does not cover the other.
          const bookingMatch = notif.link?.match(/[?&]booking=([^&]+)/);
          if (bookingMatch) {
            queryClient.invalidateQueries({ queryKey: ["direct-hire", bookingMatch[1]] });
          }
          queryClient.invalidateQueries({ queryKey: ["wallet"] });
          queryClient.invalidateQueries({ queryKey: ["wallet-transactions"] });
          // Any notification can mean a list the user is already looking at
          // (booking requests, contracts, disputes, jobs) is now stale — a
          // notification's own deep-link often points at a page that's
          // already open, and navigating to an already-open URL doesn't
          // remount/refetch it. Invalidating here is cheap: React Query only
          // actually refetches queries that are currently mounted somewhere.
          queryClient.invalidateQueries({ queryKey: ["direct-hires"] });
          queryClient.invalidateQueries({ queryKey: ["contracts"] });
          queryClient.invalidateQueries({ queryKey: ["talent-dashboard"] });
          queryClient.invalidateQueries({ queryKey: ["disputes"] });
          queryClient.invalidateQueries({ queryKey: ["jobs"] });
        } else if (envelope.type === "error") {
          setSendError(typeof envelope.data === "string" ? envelope.data : "Something went wrong sending that message.");
        }
      };
    }

    connect();

    return () => {
      stopped = true;
      clearTimeout(reconnectTimer);
      socketRef.current?.close();
      socketRef.current = null;
      setConnected(false);
      pendingQueueRef.current = [];
      setPendingSendCount(0);
    };
  }, [isAuthenticated, queryClient]);

  const sendChatMessage = useCallback((recvId, content) => {
    const payload = JSON.stringify({ recv_id: recvId, content });
    if (socketRef.current?.readyState === WebSocket.OPEN) {
      socketRef.current.send(payload);
    } else {
      pendingQueueRef.current.push(payload);
      setPendingSendCount(pendingQueueRef.current.length);
    }
    return true;
  }, []);

  // Images/voice notes go through this too — the caller uploads the file
  // via apiUpload first (fast: local disk, no processing) and passes just
  // the resulting URL here, so a dispute attachment is exactly as fast to
  // deliver as a text message.
  const sendDisputeMessage = useCallback((disputeId, content, opts = {}) => {
    const payload = JSON.stringify({
      dispute_id: disputeId,
      content,
      attachment_url: opts.attachmentUrl || "",
      attachment_type: opts.attachmentType || "",
      mentioned_user_id: opts.mentionedUserId || "",
    });
    if (socketRef.current?.readyState === WebSocket.OPEN) {
      socketRef.current.send(payload);
    } else {
      pendingQueueRef.current.push(payload);
      setPendingSendCount(pendingQueueRef.current.length);
    }
    return true;
  }, []);

  const clearUnreadMessages = useCallback(() => setUnreadByPartner({}), []);

  const clearUnreadForPartner = useCallback((partnerId) => {
    setUnreadByPartner((prev) => {
      if (!prev[partnerId]) return prev;
      const next = { ...prev };
      delete next[partnerId];
      return next;
    });
  }, []);

  const clearUnreadForDispute = useCallback((disputeId) => {
    setUnreadByDispute((prev) => {
      if (!prev[disputeId]) return prev;
      const next = { ...prev };
      delete next[disputeId];
      return next;
    });
  }, []);

  const unreadMessageCount = useMemo(
    () => Object.values(unreadByPartner).reduce((sum, n) => sum + n, 0),
    [unreadByPartner]
  );

  return (
    <RealtimeContext.Provider
      value={{
        connected,
        sendChatMessage,
        sendDisputeMessage,
        pendingSendCount,
        unreadMessageCount,
        unreadByPartner,
        unreadByDispute,
        clearUnreadMessages,
        clearUnreadForPartner,
        clearUnreadForDispute,
      }}
    >
      {children}
      <AlertDialog open={!!sendError} onOpenChange={(open) => !open && setSendError(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Message not sent</AlertDialogTitle>
            <AlertDialogDescription>{sendError}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogAction onClick={() => setSendError(null)}>Got it</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </RealtimeContext.Provider>
  );
}
