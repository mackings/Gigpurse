"use client";

import { useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useRealtime } from "@/lib/RealtimeProvider";
import ChatList from "@/components/chat/ChatList";
import ChatWindow from "@/components/chat/ChatWindow";



export default function MessagesView() {
  const searchParams = useSearchParams();
  const { user } = useCurrentUser();
  const { clearUnreadMessages } = useRealtime();
  const withParam = searchParams.get("with");
  const contractParam = searchParams.get("contract");
  const bookingParam = searchParams.get("booking");

  // A user-initiated click (or back-tap) in the list overrides whatever the
  // URL says. Until they interact, the selection is derived purely from the
  // URL (?with= directly, or ?contract= resolved via the fetch below) — no
  // effect/setState needed to keep those in sync, since it's plain
  // computation from query data on every render.

  const [manualSelectionId, setManualSelectionId] = useState(undefined);

  const { data: linkedContracts } = useQuery({
    queryKey: ["contracts", "detail", contractParam],
    queryFn: () => apiGet(`/contracts?id=${contractParam}`),
    enabled: !!contractParam && !withParam,
  });
  const linkedContract = Array.isArray(linkedContracts) ? linkedContracts[0] : linkedContracts;
  const resolvedWith =
    linkedContract && user ? (user.id === linkedContract.client_id ? linkedContract.musician_id : linkedContract.client_id) : null;

  const hasManualSelection = manualSelectionId !== undefined;
  const selectedId = hasManualSelection ? manualSelectionId : withParam || resolvedWith || null;
  const bookingId = hasManualSelection ? null : bookingParam || null;

  // Milestones live on a Contract, not a chat partner, so opening a
  // conversation the normal way (clicking a row in the list) needs its own
  // lookup — a plain user-id click has no contract id to go on otherwise,
  // which used to mean the milestones panel never mounted at all outside a
  // ?contract= deep link from a notification.
  const { data: myContracts } = useQuery({
    queryKey: ["contracts"],
    queryFn: () => apiGet("/contracts"),
  });
  function resolveContractId(counterpartId) {
    if (!counterpartId || !Array.isArray(myContracts)) return null;
    const matches = myContracts.filter((c) => c.client_id === counterpartId || c.musician_id === counterpartId);
    if (!matches.length) return null;
    return (matches.find((c) => c.status === "active") || matches[0]).id;
  }
  const contractId = contractParam || resolveContractId(selectedId);

  useEffect(() => {
    clearUnreadMessages();
  }, [clearUnreadMessages]);

  function handleSelect(userId) {
    setManualSelectionId(userId);
  }

  return (
    <div className="h-[calc(100vh-4rem)] flex bg-background">
      <div className={`${selectedId ? "hidden" : "flex"} sm:flex`}>
        <ChatList selectedId={selectedId} onSelect={handleSelect} />
      </div>
      <div className={`${selectedId ? "flex" : "hidden"} sm:flex flex-1`}>
        <ChatWindow
          key={selectedId}
          otherUserId={selectedId}
          contractId={contractId}
          bookingId={bookingId}
          onBack={() => setManualSelectionId(null)}
        />
      </div>
    </div>
  );
}
