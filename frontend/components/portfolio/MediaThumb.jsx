"use client";

import { Play, Music, Link as LinkIcon } from "lucide-react";
import { isLinkItem, domainOf } from "@/lib/portfolio-media";
import { cn } from "@/lib/utils";

// The one place that decides how a portfolio item looks as a thumbnail —
// shared by the editor's item list and the public profile's grid, so
// they never drift out of sync.
export default function MediaThumb({ item, className }) {
  if (isLinkItem(item)) {
    if (item.thumbnail_url) {
      return (
        <div className={cn("relative w-full h-full overflow-hidden bg-muted", className)}>
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={item.thumbnail_url} alt={item.title} className="w-full h-full object-cover" />
          {item.media_type !== "link" && (
            <div className="absolute inset-0 flex items-center justify-center bg-black/25">
              <div className="w-10 h-10 rounded-full bg-white/95 flex items-center justify-center shadow">
                <Play className="w-4 h-4 text-foreground fill-foreground ml-0.5" />
              </div>
            </div>
          )}
        </div>
      );
    }
    return (
      <div className={cn("w-full h-full bg-muted flex flex-col items-center justify-center gap-1.5 text-muted-foreground p-3", className)}>
        {item.media_type === "audio" ? <Music className="w-6 h-6" /> : <LinkIcon className="w-6 h-6" />}
        <span className="text-[11px] text-center truncate max-w-full">{domainOf(item.url)}</span>
      </div>
    );
  }

  if (item.media_type === "image") {
    return (
      // eslint-disable-next-line @next/next/no-img-element
      <img src={item.url} alt={item.title} className={cn("w-full h-full object-cover", className)} />
    );
  }

  if (item.media_type === "video") {
    return (
      <div className={cn("relative w-full h-full bg-black", className)}>
        <video src={item.url} className="w-full h-full object-cover" preload="metadata" muted playsInline />
        <div className="absolute inset-0 flex items-center justify-center bg-black/25">
          <div className="w-10 h-10 rounded-full bg-white/95 flex items-center justify-center shadow">
            <Play className="w-4 h-4 text-foreground fill-foreground ml-0.5" />
          </div>
        </div>
      </div>
    );
  }

  if (item.media_type === "audio") {
    return (
      <div className={cn("w-full h-full bg-gradient-to-br from-primary/15 to-primary/5 flex items-center justify-center", className)}>
        <Music className="w-7 h-7 text-primary" />
      </div>
    );
  }

  return <div className={cn("w-full h-full bg-muted", className)} />;
}
