"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Loader2, Plus, Trash2, ArrowUp, ArrowDown, ExternalLink, Upload } from "lucide-react";
import { toast } from "sonner";
import { apiGet, apiPut, apiUpload } from "@/lib/api";

const emptyItem = { title: "", description: "", url: "", media_type: "video" };

export default function TalentPortfolio() {
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [profile, setProfile] = useState(null);
  const [items, setItems] = useState([]);
  const [draft, setDraft] = useState(emptyItem);

  async function handleFileSelect(e) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    setIsUploading(true);
    try {
      const uploaded = await apiUpload("/media/upload", file);
      setDraft((prev) => ({
        ...prev,
        url: uploaded.url,
        media_type: uploaded.media_type,
        title: prev.title || file.name.replace(/\.[^/.]+$/, ""),
      }));
      toast.success("File uploaded. Fill in a title and add it below.");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsUploading(false);
    }
  }

  useEffect(() => {
    apiGet("/users/profile")
      .then((user) => {
        setProfile(user);
        setItems(user.musician_profile?.portfolio || []);
      })
      .finally(() => setIsLoading(false));
  }, []);

  function addItem() {
    if (!draft.title || !draft.url) {
      toast.error("Title and URL are required.");
      return;
    }
    setItems((prev) => [...prev, { ...draft, order: prev.length, is_featured: false }]);
    setDraft(emptyItem);
  }

  function removeItem(idx) {
    setItems((prev) => prev.filter((_, i) => i !== idx));
  }

  function move(idx, dir) {
    setItems((prev) => {
      const next = [...prev];
      const target = idx + dir;
      if (target < 0 || target >= next.length) return prev;
      [next[idx], next[target]] = [next[target], next[idx]];
      return next.map((item, i) => ({ ...item, order: i }));
    });
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
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background py-12 px-4">
      <div className="max-w-3xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-foreground tracking-tight">Your Portfolio</h1>
            <p className="text-muted-foreground">Showcase your best work to potential clients.</p>
          </div>
          <Button onClick={save} disabled={isSaving}>
            {isSaving ? <Loader2 className="w-4 h-4 animate-spin" /> : "Save changes"}
          </Button>
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Add a portfolio item</CardTitle>
            <CardDescription>Link to a video, audio track, or image showcasing your work.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <Label>Title</Label>
              <Input value={draft.title} onChange={(e) => setDraft({ ...draft, title: e.target.value })} className="mt-1.5" />
            </div>
            <div>
              <Label>Description</Label>
              <Textarea value={draft.description} onChange={(e) => setDraft({ ...draft, description: e.target.value })} className="mt-1.5" />
            </div>
            <div>
              <Label>URL</Label>
              <Input
                placeholder="https://youtube.com/watch?v=..."
                value={draft.url}
                onChange={(e) => setDraft({ ...draft, url: e.target.value })}
                className="mt-1.5"
              />
              <div className="flex items-center gap-3 mt-2">
                <span className="text-xs text-muted-foreground">or upload a file directly</span>
                <label className="inline-flex">
                  <input
                    type="file"
                    accept="image/*,audio/*,video/*"
                    onChange={handleFileSelect}
                    disabled={isUploading}
                    className="hidden"
                  />
                  <Button type="button" size="sm" variant="outline" disabled={isUploading} className="gap-1.5" asChild>
                    <span>
                      {isUploading ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Upload className="w-3.5 h-3.5" />}
                      Upload file
                    </span>
                  </Button>
                </label>
              </div>
            </div>
            <Button variant="outline" onClick={addItem} className="gap-2">
              <Plus className="w-4 h-4" />
              Add item
            </Button>
          </CardContent>
        </Card>

        <div className="space-y-3">
          {items.map((item, idx) => (
            <div key={idx} className="bg-card rounded-xl border border-border p-4 flex items-center justify-between gap-4">
              <div className="min-w-0">
                <p className="font-medium text-foreground truncate">{item.title}</p>
                <a href={item.url} target="_blank" rel="noreferrer" className="text-xs text-primary hover:underline flex items-center gap-1">
                  <ExternalLink className="w-3 h-3" />
                  {item.url}
                </a>
              </div>
              <div className="flex items-center gap-1 shrink-0">
                <Button size="icon" variant="ghost" onClick={() => move(idx, -1)}>
                  <ArrowUp className="w-4 h-4" />
                </Button>
                <Button size="icon" variant="ghost" onClick={() => move(idx, 1)}>
                  <ArrowDown className="w-4 h-4" />
                </Button>
                <Button size="icon" variant="ghost" onClick={() => removeItem(idx)} className="text-destructive hover:text-destructive">
                  <Trash2 className="w-4 h-4" />
                </Button>
              </div>
            </div>
          ))}
          {items.length === 0 && <p className="text-sm text-muted-foreground text-center py-8">No portfolio items yet.</p>}
        </div>
      </div>
    </div>
  );
}
