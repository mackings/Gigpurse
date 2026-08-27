"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Loader2, Check, MapPin, DollarSign, Video, Link as LinkIcon } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { toast } from "sonner";
import { apiGet, apiPut } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";
import AccountSettingsHeader from "@/components/account/AccountSettingsHeader";
import AccountStatusSettings from "@/components/account/AccountStatusSettings";
import AvatarUpload from "@/components/profile/AvatarUpload";
import { initials } from "@/lib/utils";

const genresList = [
  "Afrobeats", "Gospel", "R&B", "Jazz", "Classical", "Hip-Hop", "Reggae",
  "Highlife", "Juju", "Fuji", "Rock", "Pop", "Traditional", "Contemporary",
];

const instrumentsList = [
  "Piano/Keyboard", "Guitar", "Bass", "Drums", "Saxophone", "Trumpet",
  "Violin", "Cello", "Flute", "Talking Drum", "Shekere", "Voice",
];

const availabilityOptions = ["Weekdays", "Weekends", "Evenings", "Mornings", "Full-time", "Part-time"];

const socialPlatforms = [
  "instagram", "twitter", "facebook", "youtube", "tiktok", "spotify", "soundcloud", "apple_music",
];

const emptySocialLinks = Object.fromEntries(socialPlatforms.map((p) => [p, ""]));

export default function TalentOnboarding() {
  const router = useRouter();
  const { user: authUser } = useCurrentUser();
  const [step, setStep] = useState(1);
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [form, setForm] = useState({
    name: "",
    bio: "",
    location: "",
    avatar_url: "",
    stage_name: "",
    genres: [],
    instruments: [],
    experience_years: "",
    price_min: "",
    price_max: "",
    availability: [],
    intro_video_url: "",
    social_links: emptySocialLinks,
  });

  useEffect(() => {
    apiGet("/users/profile")
      .then((user) => {
        const mp = user.musician_profile || {};
        setForm((prev) => ({
          ...prev,
          name: user.name || "",
          bio: user.bio || "",
          location: user.location || "",
          avatar_url: user.avatar_url || "",
          stage_name: mp.stage_name || "",
          genres: mp.genres || [],
          instruments: mp.instruments || [],
          experience_years: mp.experience_years || "",
          price_min: mp.price_min || "",
          price_max: mp.price_max || "",
          availability: mp.availability || [],
          intro_video_url: mp.intro_video_url || "",
          social_links: { ...emptySocialLinks, ...(mp.social_links || {}) },
        }));
      })
      .catch(() => {})
      .finally(() => setIsLoading(false));
  }, []);

  function toggle(field, value) {
    setForm((prev) => ({
      ...prev,
      [field]: prev[field].includes(value) ? prev[field].filter((v) => v !== value) : [...prev[field], value],
    }));
  }

  function setSocialLink(platform, value) {
    setForm((prev) => ({ ...prev, social_links: { ...prev.social_links, [platform]: value } }));
  }

  async function handleSubmit() {
    setIsSubmitting(true);
    try {
      await apiPut("/users/profile", {
        name: form.name,
        bio: form.bio,
        location: form.location,
        avatar_url: form.avatar_url,
        musician_profile: {
          stage_name: form.stage_name,
          genres: form.genres,
          instruments: form.instruments,
          experience_years: parseInt(form.experience_years) || 0,
          price_min: parseFloat(form.price_min) || 0,
          price_max: parseFloat(form.price_max) || 0,
          availability: form.availability,
          intro_video_url: form.intro_video_url,
          social_links: form.social_links,
        },
      });
      toast.success("Profile saved!");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsSubmitting(false);
    }
  }

  const totalSteps = 4;
  const stepLabels = ["Personal", "Musical Profile", "Pricing", "Social Links"];
  const stepComplete = [
    Boolean(form.stage_name.trim() && form.bio.trim() && form.location.trim()),
    form.genres.length > 0 && form.instruments.length > 0,
    Boolean(form.price_min && form.price_max),
    Boolean(form.intro_video_url.trim() || Object.values(form.social_links).some((v) => v.trim())),
  ];
  const completionPct = Math.round((stepComplete.filter(Boolean).length / totalSteps) * 100);

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-4xl mx-auto px-4 py-12">
        <AccountSettingsHeader />

        <div className="max-w-2xl">
          <div className="mb-8">
            {/* Jump to any step directly — this lives inside the Account
                settings tab bar now, not a standalone signup wizard, so
                sequential Back/Continue paging no longer fits. Clicking a
                step is the natural way to revisit and fix an earlier one. */}
            <div className="flex items-center justify-between mb-2">
              <p className="text-sm font-medium text-foreground">Profile completeness</p>
              <p className="text-sm text-muted-foreground">{completionPct}%</p>
            </div>
            <div className="flex items-center gap-2">
              {stepLabels.map((label, idx) => {
                const n = idx + 1;
                const isCurrent = step === n;
                const isDone = stepComplete[idx];
                return (
                  <button key={label} type="button" onClick={() => setStep(n)} className="flex-1 text-left group">
                    <span
                      className={`flex items-center justify-center h-1.5 rounded-full transition-colors ${
                        isCurrent ? "bg-primary" : isDone ? "bg-status-success" : "bg-muted group-hover:bg-muted-foreground/30"
                      }`}
                    />
                    <span
                      className={`mt-1.5 flex items-center gap-1 text-xs font-medium truncate ${
                        isCurrent ? "text-foreground" : "text-muted-foreground"
                      }`}
                    >
                      {isDone && <Check className="w-3 h-3 text-status-success shrink-0" />}
                      {n}. {label}
                    </span>
                  </button>
                );
              })}
            </div>
          </div>

          <AnimatePresence mode="wait">
            {step === 1 && (
              <motion.div key="s1" initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -20 }}>
                <Card>
                  <CardHeader>
                    <CardTitle className="text-2xl">Let&apos;s set up your profile</CardTitle>
                    <CardDescription>Tell us about yourself and your musical journey</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-6">
                    <AvatarUpload
                      value={form.avatar_url}
                      onChange={(url) => setForm({ ...form, avatar_url: url })}
                      name={form.stage_name || form.name}
                    />
                    <div>
                      <Label htmlFor="stage_name">Stage Name / Artist Name</Label>
                      <Input
                        id="stage_name"
                        placeholder="Your stage name"
                        value={form.stage_name}
                        onChange={(e) => setForm({ ...form, stage_name: e.target.value })}
                        className="mt-1.5"
                      />
                    </div>
                    <div>
                      <Label htmlFor="bio">Bio</Label>
                      <Textarea
                        id="bio"
                        placeholder="Tell clients about yourself, your musical background, and what makes you unique..."
                        value={form.bio}
                        onChange={(e) => setForm({ ...form, bio: e.target.value })}
                        className="mt-1.5 min-h-[120px]"
                      />
                    </div>
                    <div>
                      <Label htmlFor="location">Location</Label>
                      <div className="relative mt-1.5">
                        <MapPin className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                        <Input
                          id="location"
                          placeholder="City, State (e.g., Lagos, Nigeria)"
                          value={form.location}
                          onChange={(e) => setForm({ ...form, location: e.target.value })}
                          className="pl-10"
                        />
                      </div>
                    </div>
                  </CardContent>
                </Card>
                {authUser && <div className="mt-6"><AccountStatusSettings user={authUser} /></div>}
              </motion.div>
            )}

            {step === 2 && (
              <motion.div key="s2" initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -20 }}>
                <Card>
                  <CardHeader>
                    <CardTitle className="text-2xl">Your Musical Profile</CardTitle>
                    <CardDescription>Help clients find you by describing your musical expertise</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-6">
                    <div>
                      <Label>Genres you perform</Label>
                      <div className="flex flex-wrap gap-2 mt-2">
                        {genresList.map((genre) => (
                          <button
                            key={genre}
                            type="button"
                            onClick={() => toggle("genres", genre)}
                            className={`px-3 py-1.5 rounded-full text-sm font-medium transition-all ${
                              form.genres.includes(genre)
                                ? "bg-primary text-primary-foreground"
                                : "bg-muted text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                            }`}
                          >
                            {genre}
                          </button>
                        ))}
                      </div>
                    </div>
                    <div>
                      <Label>Instruments you play</Label>
                      <div className="flex flex-wrap gap-2 mt-2">
                        {instrumentsList.map((instrument) => (
                          <button
                            key={instrument}
                            type="button"
                            onClick={() => toggle("instruments", instrument)}
                            className={`px-3 py-1.5 rounded-full text-sm font-medium transition-all ${
                              form.instruments.includes(instrument)
                                ? "bg-primary text-primary-foreground"
                                : "bg-muted text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                            }`}
                          >
                            {instrument}
                          </button>
                        ))}
                      </div>
                    </div>
                    <div>
                      <Label htmlFor="experience">Years of Experience</Label>
                      <Input
                        id="experience"
                        type="number"
                        min="0"
                        value={form.experience_years}
                        onChange={(e) => setForm({ ...form, experience_years: e.target.value })}
                        className="mt-1.5"
                      />
                    </div>
                  </CardContent>
                </Card>
              </motion.div>
            )}

            {step === 3 && (
              <motion.div key="s3" initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -20 }}>
                <Card>
                  <CardHeader>
                    <CardTitle className="text-2xl">Pricing & Availability</CardTitle>
                    <CardDescription>Set your rates and when you&apos;re available for gigs</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-6">
                    <div>
                      <Label>Your Price Range (per gig)</Label>
                      <div className="grid grid-cols-2 gap-4 mt-2">
                        <div className="relative">
                          <DollarSign className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                          <Input
                            type="number"
                            placeholder="Minimum"
                            value={form.price_min}
                            onChange={(e) => setForm({ ...form, price_min: e.target.value })}
                            className="pl-10"
                          />
                        </div>
                        <div className="relative">
                          <DollarSign className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                          <Input
                            type="number"
                            placeholder="Maximum"
                            value={form.price_max}
                            onChange={(e) => setForm({ ...form, price_max: e.target.value })}
                            className="pl-10"
                          />
                        </div>
                      </div>
                    </div>
                    <div>
                      <Label>Your Availability</Label>
                      <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 mt-2">
                        {availabilityOptions.map((option) => (
                          <label
                            key={option}
                            className={`flex items-center gap-3 p-3 rounded-lg border-2 cursor-pointer transition-all ${
                              form.availability.includes(option) ? "border-primary bg-accent" : "border-border hover:border-primary/40"
                            }`}
                          >
                            <Checkbox checked={form.availability.includes(option)} onCheckedChange={() => toggle("availability", option)} />
                            <span className="text-sm font-medium text-foreground">{option}</span>
                          </label>
                        ))}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </motion.div>
            )}

            {step === 4 && (
              <motion.div key="s4" initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -20 }}>
                <Card>
                  <CardHeader>
                    <CardTitle className="text-2xl">Video & Social Links</CardTitle>
                    <CardDescription>Add a video intro and connect your social profiles</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-6">
                    <div>
                      <Label className="flex items-center gap-2 mb-3">
                        <Video className="w-4 h-4 text-primary" />
                        Intro video link
                      </Label>
                      <Input
                        placeholder="https://youtube.com/watch?v=..."
                        value={form.intro_video_url}
                        onChange={(e) => setForm({ ...form, intro_video_url: e.target.value })}
                      />
                    </div>
                    <div>
                      <Label className="flex items-center gap-2 mb-3">
                        <LinkIcon className="w-4 h-4 text-primary" />
                        Social Media & Music Platforms
                      </Label>
                      <div className="grid sm:grid-cols-2 gap-3">
                        {socialPlatforms.map((platform) => (
                          <div key={platform}>
                            <Label className="text-xs text-muted-foreground capitalize">{platform.replace("_", " ")}</Label>
                            <div className="relative mt-1">
                              <LinkIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground" />
                              <Input
                                placeholder={`https://${platform}.com/yourprofile`}
                                value={form.social_links[platform]}
                                onChange={(e) => setSocialLink(platform, e.target.value)}
                                className="pl-8"
                              />
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                    <div className="flex items-center gap-4 p-4 bg-muted/50 rounded-xl border border-border">
                      <div className="w-14 h-14 rounded-full bg-primary flex items-center justify-center overflow-hidden shrink-0 text-primary-foreground font-bold">
                        {form.avatar_url ? (
                          // eslint-disable-next-line @next/next/no-img-element
                          <img src={form.avatar_url} alt="" className="w-full h-full object-cover" />
                        ) : (
                          initials(form.stage_name || form.name)
                        )}
                      </div>
                      <div>
                        <h3 className="font-bold text-foreground">{form.stage_name || "Your Stage Name"}</h3>
                        <p className="text-muted-foreground text-sm flex items-center gap-1">
                          <Check className="w-4 h-4 text-primary" />
                          {form.genres.join(", ") || "Genres not set"}
                        </p>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </motion.div>
            )}
          </AnimatePresence>

          <div className="flex justify-end mt-8">
            <Button onClick={handleSubmit} disabled={isSubmitting} className="gap-2">
              {isSubmitting ? <Loader2 className="w-4 h-4 animate-spin" /> : <>Save changes <Check className="w-4 h-4" /></>}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
