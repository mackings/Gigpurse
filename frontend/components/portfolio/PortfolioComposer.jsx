"use client";

import { useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { UploadCloud, Link2, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { apiGet, apiUpload } from "@/lib/api";

// Two ways in: drop/browse for files (uploaded straight away, one item per
// file) or paste a link (previewed via the backend's oEmbed/Open Graph
// unfurl before it's added). Either path hands finished PortfolioItem
// objects to onAdd — the parent just appends them to its list.
export default function PortfolioComposer({ onAdd, nextOrder }) {
  const [isDragging, setIsDragging] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [linkURL, setLinkURL] = useState("");
  const [isPreviewing, setIsPreviewing] = useState(false);
  const fileInputRef = useRef(null);

  async function handleFiles(fileList) {
    const files = Array.from(fileList || []).slice(0, 10);
    if (!files.length) return;
    setIsUploading(true);
    try {
      const data = await apiUpload("/media/upload", files);
      const uploaded = data.files || [{ url: data.url, media_type: data.media_type, filename: "" }];
      const items = uploaded.map((f, i) => ({
        title: (f.filename || "Untitled").replace(/\.[^/.]+$/, ""),
        description: "",
        url: f.url,
        media_type: f.media_type,
        external_url: "",
        thumbnail_url: "",
        is_featured: false,
        order: nextOrder + i,
      }));
      onAdd(items);
      toast.success(items.length > 1 ? `${items.length} files added.` : "File added.");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsUploading(false);
    }
  }

  async function handleAddLink(e) {
    e.preventDefault();
    if (!linkURL.trim()) return;
    setIsPreviewing(true);
    try {
      const preview = await apiGet(`/link-preview?url=${encodeURIComponent(linkURL.trim())}`);
      onAdd([
        {
          title: preview.title || linkURL,
          description: preview.description || "",
          url: preview.url,
          media_type: preview.media_type,
          external_url: preview.embed_url || "",
          thumbnail_url: preview.thumbnail_url || "",
          is_featured: false,
          order: nextOrder,
        },
      ]);
      setLinkURL("");
      toast.success("Link added.");
    } catch (err) {
      toast.error(err.message || "Couldn't preview that link.");
    } finally {
      setIsPreviewing(false);
    }
  }

  return (
    <div className="space-y-3">
      <div
        onDragOver={(e) => {
          e.preventDefault();
          setIsDragging(true);
        }}
        onDragLeave={() => setIsDragging(false)}
        onDrop={(e) => {
          e.preventDefault();
          setIsDragging(false);
          handleFiles(e.dataTransfer.files);
        }}
        onClick={() => fileInputRef.current?.click()}
        className={`rounded-xl border-2 border-dashed p-8 text-center cursor-pointer transition-colors ${
          isDragging ? "border-primary bg-accent" : "border-border hover:border-primary/40 hover:bg-muted/40"
        }`}
      >
        <input
          ref={fileInputRef}
          type="file"
          multiple
          accept="image/*,audio/*,video/*"
          onChange={(e) => handleFiles(e.target.files)}
          className="hidden"
        />
        {isUploading ? (
          <Loader2 className="w-6 h-6 mx-auto animate-spin text-primary" />
        ) : (
          <>
            <UploadCloud className="w-6 h-6 mx-auto text-muted-foreground" />
            <p className="text-sm font-medium text-foreground mt-2">Drop photos, videos, or audio here</p>
            <p className="text-xs text-muted-foreground mt-0.5">or click to browse — up to 10 files, 25MB each</p>
          </>
        )}
      </div>

      <form onSubmit={handleAddLink} className="flex items-center gap-2">
        <div className="relative flex-1">
          <Link2 className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="Or paste a YouTube, Vimeo, SoundCloud, Spotify link — or any project URL"
            value={linkURL}
            onChange={(e) => setLinkURL(e.target.value)}
            className="pl-9"
          />
        </div>
        <Button type="submit" variant="outline" disabled={isPreviewing || !linkURL.trim()} className="shrink-0 gap-1.5">
          {isPreviewing ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : "Add link"}
        </Button>
      </form>
    </div>
  );
}
