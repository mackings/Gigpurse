"use client";

import { useEffect, useRef, useState } from "react";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useUserInfo } from "@/hooks/use-user-info";
import { useDisputeChat } from "@/hooks/use-dispute-chat";
import { formatMessageTime, formatDayDivider } from "@/lib/format-time";
import { apiUpload } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import StatusBadge from "@/components/ui/status-badge";
import ResolveDisputeModal from "@/components/disputes/ResolveDisputeModal";
import { ArrowLeft, Image as ImageIcon, Mic, Square, Send, ShieldCheck, Loader2, AtSign, Gavel } from "lucide-react";
import { toast } from "sonner";

function useVoiceRecorder(onDone) {
  const [recording, setRecording] = useState(false);
  const [seconds, setSeconds] = useState(0);
  const recorderRef = useRef(null);
  const chunksRef = useRef([]);
  const timerRef = useRef(null);

  async function start() {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const recorder = new MediaRecorder(stream);
      chunksRef.current = [];
      recorder.ondataavailable = (e) => e.data.size > 0 && chunksRef.current.push(e.data);
      recorder.onstop = () => {
        stream.getTracks().forEach((t) => t.stop());
        clearInterval(timerRef.current);
        const blob = new Blob(chunksRef.current, { type: recorder.mimeType || "audio/webm" });
        setSeconds(0);
        onDone(blob);
      };
      recorder.start();
      recorderRef.current = recorder;
      setRecording(true);
      setSeconds(0);
      timerRef.current = setInterval(() => setSeconds((s) => s + 1), 1000);
    } catch {
      toast.error("Couldn't access your microphone.");
    }
  }

  function stop() {
    recorderRef.current?.stop();
    setRecording(false);
  }

  return { recording, seconds, start, stop };
}

function MessageBubble({ msg, isMine, senderLabel, isModeratorSender, isMentioned }) {
  if (msg.is_system) {
    return (
      <div className="flex justify-center my-2">
        <span className="text-[11px] text-center max-w-[85%] text-muted-foreground bg-muted px-3 py-1.5 rounded-full">
          {msg.content}
        </span>
      </div>
    );
  }

  return (
    <div className={`flex ${isMine ? "justify-end" : "justify-start"} mb-1.5`}>
      <div className="max-w-[78%] sm:max-w-[65%]">
        {!isMine && (
          <p className="text-[11px] text-muted-foreground mb-0.5 px-1 flex items-center gap-1">
            {senderLabel}
            {isModeratorSender && (
              <span className="text-primary font-medium flex items-center gap-0.5">
                <ShieldCheck className="w-3 h-3" />
                Moderator
              </span>
            )}
          </p>
        )}
        <div
          className={`rounded-2xl px-3.5 py-2 text-sm ${
            isMine ? "bg-primary text-primary-foreground font-medium" : "bg-muted text-foreground"
          }`}
        >
          {isMentioned && (
            <p className={`text-[11px] mb-1 flex items-center gap-1 ${isMine ? "text-primary-foreground/80" : "text-primary"}`}>
              <AtSign className="w-3 h-3" />
              tagged for proof
            </p>
          )}
          {msg.attachment_type === "image" && (
            /* eslint-disable-next-line @next/next/no-img-element */
            <img
              src={msg.attachment_url}
              alt="Shared screenshot"
              className="rounded-lg max-w-full max-h-64 mb-1.5 cursor-pointer"
              onClick={() => window.open(msg.attachment_url, "_blank")}
            />
          )}
          {msg.attachment_type === "audio" && <audio controls src={msg.attachment_url} className="max-w-full mb-1.5" />}
          {msg.content && <span className="break-words">{msg.content}</span>}
          <div className={`text-[10px] mt-1 ${isMine ? "text-primary-foreground/70" : "text-muted-foreground"}`}>
            {formatMessageTime(msg.timestamp)}
          </div>
        </div>
      </div>
    </div>
  );
}

export default function DisputeChatRoom({ disputeId, onBack }) {
  const { user } = useCurrentUser();
  const { dispute, isLoadingDispute, messages, sendMessage, join, resolve } = useDisputeChat(disputeId);
  const clientInfo = useUserInfo(dispute?.client_id);
  const musicianInfo = useUserInfo(dispute?.musician_id);
  const moderatorInfo = useUserInfo(dispute?.moderator_id);
  const [draft, setDraft] = useState("");
  const [tagTarget, setTagTarget] = useState("");
  const [isJoining, setIsJoining] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [resolveOpen, setResolveOpen] = useState(false);
  const fileInputRef = useRef(null);
  const bottomRef = useRef(null);

  const isModerator = user?.role === "admin" || user?.role === "moderator";
  const isAssignedModerator = dispute?.moderator_id === user?.id;
  const isBlocked = !dispute?.moderator_id && !isModerator;
  const isResolved = dispute?.status === "resolved" || dispute?.status === "closed";

  const recorder = useVoiceRecorder(async (blob) => {
    setIsUploading(true);
    try {
      const file = new File([blob], `voice-note.${blob.type.includes("webm") ? "webm" : "m4a"}`, { type: blob.type });
      const { url, media_type } = await apiUpload("/media/upload", file);
      sendMessage("", { attachmentUrl: url, attachmentType: media_type === "audio" ? "audio" : "audio", mentionedUserId: tagTarget });
      setTagTarget("");
    } catch (err) {
      toast.error(err.message || "Couldn't send that voice note.");
    } finally {
      setIsUploading(false);
    }
  });

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages?.length]);

  function nameFor(userId) {
    if (userId === dispute?.client_id) return clientInfo?.name || "Client";
    if (userId === dispute?.musician_id) return musicianInfo?.name || "Talent";
    if (userId === dispute?.moderator_id) return moderatorInfo?.name || "Moderator";
    return "Someone";
  }

  async function handleJoin() {
    setIsJoining(true);
    try {
      await join();
      toast.success("You've joined this dispute as moderator.");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsJoining(false);
    }
  }

  function handleSend(e) {
    e.preventDefault();
    if (!draft.trim()) return;
    sendMessage(draft.trim(), { mentionedUserId: tagTarget });
    setDraft("");
    setTagTarget("");
  }

  async function handleImagePick(e) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    setIsUploading(true);
    try {
      const { url } = await apiUpload("/media/upload", file);
      sendMessage("", { attachmentUrl: url, attachmentType: "image", mentionedUserId: tagTarget });
      setTagTarget("");
    } catch (err) {
      toast.error(err.message || "Couldn't send that image.");
    } finally {
      setIsUploading(false);
    }
  }

  if (isLoadingDispute || !dispute) {
    return (
      <div className="flex-1 flex items-center justify-center bg-muted/20">
        <Loader2 className="w-6 h-6 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col h-full bg-background min-w-0">
      <div className="p-3 sm:p-4 border-b border-border flex items-center gap-3 justify-between bg-background">
        <div className="flex items-center gap-3 min-w-0">
          {onBack && (
            <Button variant="ghost" size="icon" className="sm:hidden shrink-0 -ml-1" onClick={onBack}>
              <ArrowLeft className="w-5 h-5" />
            </Button>
          )}
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <h2 className="font-semibold text-foreground truncate">Dispute</h2>
              <StatusBadge status={dispute.status} />
            </div>
            <p className="text-xs text-muted-foreground truncate">
              {clientInfo?.name || "Client"} &amp; {musicianInfo?.name || "Talent"}
              {dispute.moderator_id && ` · moderated by ${moderatorInfo?.name || "a moderator"}`}
            </p>
          </div>
        </div>
        {isModerator && dispute.status === "open" && (
          <div className="flex items-center gap-2 shrink-0">
            {!dispute.moderator_id && (
              <Button size="sm" onClick={handleJoin} disabled={isJoining} className="gap-1.5">
                {isJoining ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <ShieldCheck className="w-3.5 h-3.5" />}
                Join as moderator
              </Button>
            )}
            <Button size="sm" variant="outline" onClick={() => setResolveOpen(true)} className="gap-1.5">
              <Gavel className="w-3.5 h-3.5" />
              Resolve
            </Button>
          </div>
        )}
      </div>

      <div className="flex-1 overflow-y-auto p-3 sm:p-4 space-y-1 bg-muted/10">
        {messages.map((msg, idx) => {
          const day = msg.timestamp ? new Date(msg.timestamp).toDateString() : null;
          const prevDay = idx > 0 && messages[idx - 1].timestamp ? new Date(messages[idx - 1].timestamp).toDateString() : null;
          const showDivider = day && day !== prevDay;
          return (
            <div key={msg.id || idx}>
              {showDivider && (
                <div className="flex justify-center my-3">
                  <span className="text-[11px] font-medium text-muted-foreground bg-muted px-2.5 py-1 rounded-full">
                    {formatDayDivider(msg.timestamp)}
                  </span>
                </div>
              )}
              <MessageBubble
                msg={msg}
                isMine={msg.sender_id === user?.id}
                senderLabel={nameFor(msg.sender_id)}
                isModeratorSender={msg.sender_id === dispute.moderator_id}
                isMentioned={msg.mentioned_user_id === user?.id}
              />
            </div>
          );
        })}
        <div ref={bottomRef} />
      </div>

      {isResolved ? (
        <div className="p-3 sm:p-4 border-t border-border text-sm text-center text-muted-foreground bg-muted/30">
          This dispute is resolved{dispute.winner_id && ` in favor of ${nameFor(dispute.winner_id)}`}. The conversation is closed.
        </div>
      ) : isBlocked ? (
        <div className="p-3 sm:p-4 border-t border-border text-sm text-center text-muted-foreground bg-muted/30">
          Waiting for a moderator to join before you two can chat here.
        </div>
      ) : (
        <form onSubmit={handleSend} className="border-t border-border bg-background">
          {isModerator && (
            <div className="flex items-center gap-1.5 px-3 sm:px-4 pt-2 text-xs text-muted-foreground">
              <AtSign className="w-3.5 h-3.5" />
              Tag for proof:
              <button
                type="button"
                onClick={() => setTagTarget((t) => (t === dispute.client_id ? "" : dispute.client_id))}
                className={`px-2 py-0.5 rounded-full border ${tagTarget === dispute.client_id ? "bg-primary text-primary-foreground border-primary" : "border-border"}`}
              >
                {clientInfo?.name || "Client"}
              </button>
              <button
                type="button"
                onClick={() => setTagTarget((t) => (t === dispute.musician_id ? "" : dispute.musician_id))}
                className={`px-2 py-0.5 rounded-full border ${tagTarget === dispute.musician_id ? "bg-primary text-primary-foreground border-primary" : "border-border"}`}
              >
                {musicianInfo?.name || "Talent"}
              </button>
            </div>
          )}
          <div className="p-3 sm:p-4 flex gap-2 items-center">
            <input ref={fileInputRef} type="file" accept="image/*" className="hidden" onChange={handleImagePick} />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="shrink-0 rounded-full"
              disabled={isUploading || recorder.recording}
              onClick={() => fileInputRef.current?.click()}
              title="Share a screenshot"
            >
              <ImageIcon className="w-4 h-4" />
            </Button>
            <Button
              type="button"
              variant={recorder.recording ? "destructive" : "ghost"}
              size="icon"
              className="shrink-0 rounded-full"
              disabled={isUploading}
              onClick={recorder.recording ? recorder.stop : recorder.start}
              title={recorder.recording ? "Stop recording" : "Record a voice note"}
            >
              {recorder.recording ? <Square className="w-4 h-4" /> : <Mic className="w-4 h-4" />}
            </Button>
            {recorder.recording ? (
              <div className="flex-1 flex items-center gap-2 text-sm text-destructive">
                <span className="w-2 h-2 rounded-full bg-destructive animate-pulse" />
                Recording {String(Math.floor(recorder.seconds / 60)).padStart(2, "0")}:{String(recorder.seconds % 60).padStart(2, "0")}
              </div>
            ) : (
              <Input
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                placeholder={isUploading ? "Sending..." : "Type a message..."}
                disabled={isUploading}
                className="rounded-full"
              />
            )}
            <Button type="submit" size="icon" className="shrink-0 rounded-full" disabled={isUploading || recorder.recording || !draft.trim()}>
              {isUploading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
            </Button>
          </div>
        </form>
      )}

      <ResolveDisputeModal
        dispute={dispute}
        clientName={clientInfo?.name || "Client"}
        musicianName={musicianInfo?.name || "Talent"}
        open={resolveOpen}
        onOpenChange={setResolveOpen}
        onResolve={resolve}
      />
    </div>
  );
}
