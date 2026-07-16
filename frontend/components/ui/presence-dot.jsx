"use client";

import { Circle, CircleDashed, Ban } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

// "hidden" is deliberately not a renderable value here — an observer sees
// "offline" instead, which is the entire point of that setting. Only the
// account owner ever sees their own "hidden" state, in account settings.
const PRESENCE = {
  online: { icon: Circle, className: "text-emerald-500 fill-emerald-500", label: "Online now" },
  offline: { icon: CircleDashed, className: "text-muted-foreground", label: "Offline" },
  disabled: { icon: Ban, className: "text-rose-500", label: "This account is temporarily disabled" },
};

export default function PresenceDot({ status, size = "sm", className }) {
  const entry = PRESENCE[status];
  if (!entry) return null;
  const Icon = entry.icon;
  const dim = size === "lg" ? "w-3.5 h-3.5" : "w-2.5 h-2.5";

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={cn("inline-flex shrink-0 cursor-default", className)}>
          <Icon className={cn(dim, entry.className)} />
        </span>
      </TooltipTrigger>
      <TooltipContent>{entry.label}</TooltipContent>
    </Tooltip>
  );
}
