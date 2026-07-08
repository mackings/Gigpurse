"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { apiGet, apiPut } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Loader2 } from "lucide-react";
import PortfolioComposer from "@/components/portfolio/PortfolioComposer";
import PortfolioItemCard from "@/components/portfolio/PortfolioItemCard";
import PortfolioLightbox from "@/components/portfolio/PortfolioLightbox";
import ShareLinkButton from "@/components/ShareLinkButton";

export default function TalentPortfolio() {
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [profile, setProfile] = useState(null);
  const [items, setItems] = useState([]);
  const [previewItem, setPreviewItem] = useState(null);
  const [profileURL, setProfileURL] = useState("");

  useEffect(() => {
    apiGet("/users/profile")
      .then((user) => {
        setProfile(user);
        setItems(user.musician_profile?.portfolio || []);
        if (typeof window !== "undefined") {
          setProfileURL(`${window.location.origin}/talent/${user.id}`);
        }
      })
      .finally(() => setIsLoading(false));
  }, []);

  function addItems(newItems) {
    setItems((prev) => [...prev, ...newItems]);
  }

  function updateItem(index, next) {
    setItems((prev) => prev.map((it, i) => (i === index ? next : it)));
  }

  function removeItem(index) {
    setItems((prev) => prev.filter((_, i) => i !== index).map((it, i) => ({ ...it, order: i })));
  }

  function moveItem(index, dir) {
    return () => {
      setItems((prev) => {
        const next = [...prev];
        const target = index + dir;
        if (target < 0 || target >= next.length) return prev;
        [next[index], next[target]] = [next[target], next[index]];
        return next.map((it, i) => ({ ...it, order: i }));
      });
    };
  }

  async function save() {
    setIsSaving(true);
    try {
      await apiPut("/users/profile", {
        name: profile.name,
        bio: profile.bio,
        location: profile.location,
        musician_profile: { ...profile.musician_profile, portfolio: items },
      });
      toast.success("Portfolio saved!");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsSaving(false);
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  const featuredCount = items.filter((i) => i.is_featured).length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h2 className="text-lg font-semibold text-foreground tracking-tight">Your Portfolio</h2>
          <p className="text-muted-foreground text-sm">
            Show clients your best work — photos, videos, tracks, anything you&apos;re proud of.
            {featuredCount > 0 && ` ${featuredCount} featured.`}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {profileURL && <ShareLinkButton url={profileURL} label="Share profile" />}
          <Button onClick={save} disabled={isSaving}>
            {isSaving ? <Loader2 className="w-4 h-4 animate-spin" /> : "Save changes"}
          </Button>
        </div>
      </div>

      <PortfolioComposer onAdd={addItems} nextOrder={items.length} />

      {items.length > 0 ? (
        <div className="space-y-3">
          {items.map((item, idx) => (
            <PortfolioItemCard
              key={idx}
              item={item}
              onChange={(next) => updateItem(idx, next)}
              onRemove={() => removeItem(idx)}
              onMove={(dir) => moveItem(idx, dir)}
              onPreview={setPreviewItem}
              canMoveUp={idx > 0}
              canMoveDown={idx < items.length - 1}
            />
          ))}
        </div>
      ) : (
        <p className="text-sm text-muted-foreground text-center py-12">No portfolio items yet — add your first one above.</p>
      )}

      <PortfolioLightbox item={previewItem} open={!!previewItem} onOpenChange={(o) => !o && setPreviewItem(null)} />
    </div>
  );
}
