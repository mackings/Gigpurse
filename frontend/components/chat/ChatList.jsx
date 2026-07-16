"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useRealtime } from "@/lib/RealtimeProvider";
import { useUserInfo } from "@/hooks/use-user-info";
import { useUserStatus } from "@/hooks/use-user-status";
import PresenceDot from "@/components/ui/presence-dot";
import { formatMessageTime } from "@/lib/format-time";
import { Input } from "@/components/ui/input";
import { MessageCircle, Search } from "lucide-react";

function partnerIdFor(msg, myId) {
  return msg.sender_id === myId ? msg.recv_id : msg.sender_id;
}

function ConversationRow({ partnerId, lastMessage, selected, unread, onSelect }) {
  const partner = useUserInfo(partnerId);
  const status = useUserStatus(partnerId);
  const label = partner?.name || `User ${partnerId.slice(-6)}`;

  return (
    <button
      onClick={() => onSelect(partnerId)}
      className={`w-full text-left px-4 py-3 flex items-center gap-3 border-b border-border/60 hover:bg-accent transition-colors ${
        selected ? "bg-accent" : ""
      }`}
    >
      <div className="relative shrink-0">
        <div className="w-12 h-12 rounded-full bg-primary flex items-center justify-center text-primary-foreground text-sm font-semibold">
          {label.charAt(0).toUpperCase()}
        </div>
        <PresenceDot status={status} className="absolute -bottom-0.5 -right-0.5 bg-background rounded-full p-0.5" />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between gap-2">
          <span className={`text-sm truncate ${unread > 0 ? "font-semibold text-foreground" : "font-medium text-foreground"}`}>
            {label}
          </span>
          <span className={`text-xs shrink-0 ${unread > 0 ? "text-primary font-medium" : "text-muted-foreground"}`}>
            {formatMessageTime(lastMessage.timestamp)}
          </span>
        </div>
        <div className="flex items-center justify-between gap-2 mt-0.5">
          <p className={`text-sm truncate ${unread > 0 ? "text-foreground" : "text-muted-foreground"}`}>{lastMessage.content}</p>
          {unread > 0 && (
            <span className="shrink-0 min-w-[20px] h-5 px-1.5 rounded-full bg-primary text-primary-foreground text-[11px] font-semibold flex items-center justify-center">
              {unread > 9 ? "9+" : unread}
            </span>
          )}
        </div>
      </div>
    </button>
  );
}

export default function ChatList({ selectedId, onSelect }) {
  const { user } = useCurrentUser();
  const { unreadByPartner } = useRealtime();
  const [search, setSearch] = useState("");
  const { data: recent, isLoading } = useQuery({
    queryKey: ["chats-recent"],
    queryFn: () => apiGet("/chats/recent"),
    enabled: !!user,
  });

  const conversations = (recent || [])
    .map((msg) => ({ partnerId: partnerIdFor(msg, user?.id), lastMessage: msg }))
    .filter((c, idx, arr) => arr.findIndex((x) => x.partnerId === c.partnerId) === idx)
    .filter(
      (c) =>
        !search ||
        c.partnerId.toLowerCase().includes(search.toLowerCase()) ||
        c.lastMessage.content.toLowerCase().includes(search.toLowerCase())
    );

  return (
    <div className="w-full sm:w-80 md:w-96 border-r border-border flex flex-col h-full shrink-0 bg-background">
      <div className="p-4 border-b border-border space-y-3">
        <h1 className="text-xl font-bold text-foreground">Chats</h1>
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search conversations"
            className="pl-9 bg-muted/50 border-transparent"
          />
        </div>
      </div>
      <div className="flex-1 overflow-y-auto">
        {isLoading ? (
          <p className="p-4 text-sm text-muted-foreground">Loading...</p>
        ) : conversations.length ? (
          conversations.map(({ partnerId, lastMessage }) => (
            <ConversationRow
              key={partnerId}
              partnerId={partnerId}
              lastMessage={lastMessage}
              selected={selectedId === partnerId}
              unread={unreadByPartner[partnerId] || 0}
              onSelect={onSelect}
            />
          ))
        ) : (
          <div className="p-6 text-center text-muted-foreground">
            <MessageCircle className="w-8 h-8 mx-auto mb-2" />
            <p className="text-sm">{search ? "No matching conversations." : "No conversations yet."}</p>
          </div>
        )}
      </div>
    </div>
  );
}
