"use client";

import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useRealtime } from "@/lib/RealtimeProvider";
import { useMilestones } from "@/hooks/use-milestones";
import { useUserInfo } from "@/hooks/use-user-info";
import { formatMessageTime, formatDayDivider } from "@/lib/format-time";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import MilestoneList from "@/components/milestones/MilestoneList";
import CreateMilestonesModal from "@/components/milestones/CreateMilestonesModal";
import BookingRequestPanel from "@/components/booking/BookingRequestPanel";
import FirstMessageDialog from "@/components/chat/FirstMessageDialog";
import { ArrowLeft, ChevronDown, ChevronUp, Loader2, Plus, Send } from "lucide-react";

function MilestonePanel({ contractId }) {
  const { user } = useCurrentUser();
  const [expanded, setExpanded] = useState(true);

  const { data: contracts } = useQuery({
    queryKey: ["contracts", "detail", contractId],
    queryFn: () => apiGet(`/contracts?id=${contractId}`),
    enabled: !!contractId,
  });
  const contract = Array.isArray(contracts) ? contracts[0] : contracts;
  const role = contract && user ? (user.id === contract.client_id ? "client" : "musician") : null;

  const { milestones, propose, accept, reject, counter, fund, release } = useMilestones(contractId);

  if (!contractId || !contract || !user) return null;

  return (
    <div className="border-b border-border bg-muted/30">
      <button
        type="button"
        onClick={() => setExpanded((v) => !v)}
        className="w-full flex items-center justify-between px-4 py-2.5 text-sm font-medium text-foreground"
      >
        <span>Milestones & escrow {milestones.length > 0 && `(${milestones.length})`}</span>
        {expanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
      </button>
      {expanded && (
        <div className="px-4 pb-4 space-y-3">
          <div className="flex justify-end">
            <CreateMilestonesModal
              trigger={
                <Button size="sm" variant="outline" className="gap-1.5">
                  <Plus className="w-3.5 h-3.5" />
                  Propose milestone
                </Button>
              }
              onCreate={propose}
            />
          </div>
          <MilestoneList
            milestones={milestones}
            role={role}
            currentUserId={user.id}
            onAccept={accept}
            onReject={reject}
            onCounter={counter}
            onFund={fund}
            onRelease={release}
          />
        </div>
      )}
    </div>
  );
}

export default function ChatWindow({ otherUserId, contractId, bookingId, onBack }) {
  const { user } = useCurrentUser();
  const { connected, sendChatMessage, pendingSendCount, clearUnreadForPartner } = useRealtime();
  const otherUser = useUserInfo(otherUserId);
  const otherUserLabel = otherUser?.name || (otherUserId ? `User ${otherUserId.slice(-6)}` : "");
  const [draft, setDraft] = useState("");
  const [showSafetyDialog, setShowSafetyDialog] = useState(false);
  const bottomRef = useRef(null);

  // The RealtimeProvider's shared socket appends live messages straight into
  // this query's cache (queryKey ["chat-history", otherUserId]), so this is
  // the single source of truth — no local socket, no local message state.
  const { data: messages, isLoading } = useQuery({
    queryKey: ["chat-history", otherUserId],
    queryFn: () => apiGet(`/chats/history?user_id=${otherUserId}`),
    enabled: !!otherUserId,
  });

  useEffect(() => {
    if (otherUserId) clearUnreadForPartner(otherUserId);
  }, [otherUserId, clearUnreadForPartner]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages?.length]);

  function actuallySend() {
    if (sendChatMessage(otherUserId, draft)) {
      setDraft("");
    }
  }

  function handleSend(e) {
    e.preventDefault();
    if (!draft.trim() || isLoading) return;
    // "First message to this contact" is just "no chat history exists yet" —
    // once one message goes out, history is non-empty and this never fires
    // again. Guard against both [] and a null/undefined response.
    if (!messages || messages.length === 0) {
      setShowSafetyDialog(true);
      return;
    }
    actuallySend();
  }

  if (!otherUserId) {
    return (
      <div className="hidden sm:flex flex-1 items-center justify-center text-muted-foreground bg-muted/20">
        Select a conversation to start messaging
      </div>
    );
  }

  let lastDay = null;

  return (
    <div className="flex-1 flex flex-col h-full bg-background">
      <div className="p-3 sm:p-4 border-b border-border flex items-center gap-3 justify-between bg-background">
        <div className="flex items-center gap-3 min-w-0">
          <Button variant="ghost" size="icon" className="sm:hidden shrink-0 -ml-1" onClick={onBack}>
            <ArrowLeft className="w-5 h-5" />
          </Button>
          <div className="w-9 h-9 rounded-full bg-primary flex items-center justify-center text-primary-foreground text-xs font-semibold shrink-0">
            {otherUserLabel.charAt(0).toUpperCase()}
          </div>
          <h2 className="font-semibold text-foreground truncate">{otherUserLabel}</h2>
        </div>
        <span className={`text-xs px-2 py-0.5 rounded-full shrink-0 ${connected ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400" : "bg-muted text-muted-foreground"}`}>
          {connected ? "Live" : pendingSendCount > 0 ? `Reconnecting — ${pendingSendCount} message${pendingSendCount > 1 ? "s" : ""} queued` : "Connecting..."}
        </span>
      </div>

      <BookingRequestPanel otherUserId={otherUserId} bookingId={bookingId} />
      <MilestonePanel contractId={contractId} />

      <div className="flex-1 overflow-y-auto p-3 sm:p-4 space-y-1 bg-muted/10">
        {isLoading ? (
          <div className="flex justify-center py-12">
            <Loader2 className="w-6 h-6 animate-spin text-primary" />
          </div>
        ) : messages?.length ? (
          messages.map((msg, idx) => {
            const isMine = msg.sender_id === user?.id;
            const day = msg.timestamp ? new Date(msg.timestamp).toDateString() : null;
            const showDivider = day && day !== lastDay;
            lastDay = day;
            return (
              <div key={msg.id || idx}>
                {showDivider && (
                  <div className="flex justify-center my-3">
                    <span className="text-[11px] font-medium text-muted-foreground bg-muted px-2.5 py-1 rounded-full">
                      {formatDayDivider(msg.timestamp)}
                    </span>
                  </div>
                )}
                <div className={`flex ${isMine ? "justify-end" : "justify-start"} mb-1.5`}>
                  <div
                    className={`max-w-[78%] sm:max-w-[65%] rounded-2xl px-3.5 py-2 text-sm flex items-end gap-2 flex-wrap ${
                      isMine ? "bg-primary text-primary-foreground font-medium" : "bg-muted text-foreground"
                    }`}
                  >
                    <span className="break-words">{msg.content}</span>
                    <span className={`text-[10px] shrink-0 ml-auto ${isMine ? "text-primary-foreground/70" : "text-muted-foreground"}`}>
                      {formatMessageTime(msg.timestamp)}
                    </span>
                  </div>
                </div>
              </div>
            );
          })
        ) : (
          <p className="text-center text-muted-foreground text-sm py-12">No messages yet. Say hello!</p>
        )}
        <div ref={bottomRef} />
      </div>

      <form onSubmit={handleSend} className="p-3 sm:p-4 border-t border-border flex gap-2 bg-background">
        <Input value={draft} onChange={(e) => setDraft(e.target.value)} placeholder="Type a message..." className="rounded-full" />
        <Button type="submit" size="icon" className="shrink-0 rounded-full" disabled={isLoading}>
          <Send className="w-4 h-4" />
        </Button>
      </form>

      <FirstMessageDialog
        open={showSafetyDialog}
        onOpenChange={setShowSafetyDialog}
        onConfirm={actuallySend}
        recipientName={otherUserLabel}
      />
    </div>
  );
}
