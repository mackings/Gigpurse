"use client";

import { useState } from "react";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Star, Trash2, ArrowUp, ArrowDown, Maximize2 } from "lucide-react";
import MediaThumb from "@/components/portfolio/MediaThumb";
import { cn } from "@/lib/utils";

export default function PortfolioItemCard({ item, onChange, onRemove, onMove, onPreview, canMoveUp, canMoveDown }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="bg-card rounded-xl border border-border overflow-hidden flex flex-col sm:flex-row">
      <button
        type="button"
        onClick={() => onPreview(item)}
        className="relative w-full sm:w-40 h-40 shrink-0 group"
      >
        <MediaThumb item={item} className="rounded-none" />
        <div className="absolute inset-0 bg-black/0 group-hover:bg-black/20 transition-colors flex items-center justify-center opacity-0 group-hover:opacity-100">
          <Maximize2 className="w-5 h-5 text-white" />
        </div>
      </button>

      <div className="flex-1 p-4 space-y-2 min-w-0">
        <div className="flex items-start gap-2">
          <Input
            value={item.title}
            onChange={(e) => onChange({ ...item, title: e.target.value })}
            placeholder="Give this piece a title"
            className="font-medium"
          />
          <Button
            type="button"
            size="icon"
            variant={item.is_featured ? "default" : "outline"}
            onClick={() => onChange({ ...item, is_featured: !item.is_featured })}
            className="shrink-0"
            title={item.is_featured ? "Remove from featured" : "Feature this piece"}
          >
            <Star className={cn("w-4 h-4", item.is_featured && "fill-current")} />
          </Button>
        </div>

        {expanded ? (
          <Textarea
            value={item.description}
            onChange={(e) => onChange({ ...item, description: e.target.value })}
            placeholder="What was this project — the gig, the role, the moment?"
            className="text-sm"
            rows={2}
            onBlur={() => !item.description && setExpanded(false)}
          />
        ) : (
          <button
            type="button"
            onClick={() => setExpanded(true)}
            className="text-sm text-muted-foreground text-left hover:text-foreground truncate w-full"
          >
            {item.description || "Add a description…"}
          </button>
        )}

        <div className="flex items-center justify-between pt-1">
          <span className="text-xs text-muted-foreground truncate capitalize">{item.media_type}</span>
          <div className="flex items-center gap-1 shrink-0">
            <Button type="button" size="icon" variant="ghost" disabled={!canMoveUp} onClick={onMove(-1)}>
              <ArrowUp className="w-4 h-4" />
            </Button>
            <Button type="button" size="icon" variant="ghost" disabled={!canMoveDown} onClick={onMove(1)}>
              <ArrowDown className="w-4 h-4" />
            </Button>
            <Button type="button" size="icon" variant="ghost" onClick={onRemove} className="text-destructive hover:text-destructive">
              <Trash2 className="w-4 h-4" />
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
