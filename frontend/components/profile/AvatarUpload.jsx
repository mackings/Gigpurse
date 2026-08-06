"use client";

import { useRef, useState } from "react";
import { apiUpload } from "@/lib/api";
import { Camera, Loader2 } from "lucide-react";
import { toast } from "sonner";

// Circular photo picker used on both profile-completion flows (client
// details, talent onboarding) — reuses the same generic /media/upload
// endpoint every other file upload in the app already goes through.
// Falls back to an initials avatar (matching every other avatar rendered
// across the app) until a photo is set.
export default function AvatarUpload({ value, onChange, name, size = 96 }) {
  const [isUploading, setIsUploading] = useState(false);
  const fileInputRef = useRef(null);

  async function handleFile(e) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    if (!file.type.startsWith("image/")) {
      toast.error("Please choose an image file.");
      return;
    }
    setIsUploading(true);
    try {
      const { url } = await apiUpload("/media/upload", file);
      onChange(url);
    } catch (err) {
      toast.error(err.message || "Couldn't upload that photo.");
    } finally {
      setIsUploading(false);
    }
  }

  const initial = (name || "?").charAt(0).toUpperCase();

  return (
    <div className="flex items-center gap-4">
      <div className="relative shrink-0" style={{ width: size, height: size }}>
        <div
          className="w-full h-full rounded-full bg-primary flex items-center justify-center text-primary-foreground font-bold overflow-hidden ring-4 ring-card shadow-sm"
          style={{ fontSize: size / 2.5 }}
        >
          {value ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={value} alt="Profile" className="w-full h-full object-cover" />
          ) : (
            initial
          )}
        </div>
        <button
          type="button"
          onClick={() => fileInputRef.current?.click()}
          disabled={isUploading}
          title="Change photo"
          className="absolute bottom-0 right-0 w-8 h-8 rounded-full bg-card border border-border shadow-sm flex items-center justify-center hover:bg-muted transition-colors"
        >
          {isUploading ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Camera className="w-3.5 h-3.5" />}
        </button>
        <input ref={fileInputRef} type="file" accept="image/*" className="hidden" onChange={handleFile} />
      </div>
      <div>
        <p className="text-sm font-medium text-foreground">Profile photo</p>
        <p className="text-xs text-muted-foreground">Shown on your public profile and to clients/talent you interact with.</p>
      </div>
    </div>
  );
}
