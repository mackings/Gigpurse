"use client";

import { Dialog, DialogContent, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { ExternalLink, Star } from "lucide-react";
import { isLinkItem, isEmbeddable, domainOf } from "@/lib/portfolio-media";

export default function PortfolioLightbox({ item, open, onOpenChange }) {
  if (!item) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl p-0 overflow-hidden gap-0">
        <div className="bg-black flex items-center justify-center">{renderMedia(item)}</div>
        <div className="p-5 space-y-2">
          <div className="flex items-start justify-between gap-3">
            <DialogTitle className="text-lg">{item.title}</DialogTitle>
            {item.is_featured && (
              <Badge className="gap-1 shrink-0">
                <Star className="w-3 h-3 fill-current" />
                Featured
              </Badge>
            )}
          </div>
          {item.description && <DialogDescription>{item.description}</DialogDescription>}
          {isLinkItem(item) && (
            <a
              href={item.url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
            >
              <ExternalLink className="w-3 h-3" />
              View on {domainOf(item.url)}
            </a>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function renderMedia(item) {
  if (isEmbeddable(item)) {
    return (
      <div className="w-full aspect-video">
        <iframe
          src={item.external_url}
          className="w-full h-full"
          allow="autoplay; encrypted-media; picture-in-picture"
          allowFullScreen
        />
      </div>
    );
  }

  if (isLinkItem(item)) {
    return item.thumbnail_url ? (
      // eslint-disable-next-line @next/next/no-img-element
      <img src={item.thumbnail_url} alt={item.title} className="w-full max-h-[70vh] object-contain" />
    ) : (
      <div className="w-full aspect-video flex items-center justify-center text-muted-foreground text-sm">
        No preview available
      </div>
    );
  }

  if (item.media_type === "image") {
    // eslint-disable-next-line @next/next/no-img-element
    return <img src={item.url} alt={item.title} className="w-full max-h-[70vh] object-contain" />;
  }

  if (item.media_type === "video") {
    return <video src={item.url} controls autoPlay className="w-full max-h-[70vh]" />;
  }

  if (item.media_type === "audio") {
    return (
      <div className="w-full py-16 px-8">
        <audio src={item.url} controls autoPlay className="w-full" />
      </div>
    );
  }

  return null;
}
