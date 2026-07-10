"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { ArrowLeft, ArrowRight, Loader2, Disc3, MapPin, DollarSign, Check, Video, Link as LinkIcon } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { toast } from "sonner";
import { apiGet, apiPut } from "@/lib/api";

const genresList = [
  "Afrobeats", "Gospel", "R&B", "Jazz", "Classical", "Hip-Hop", "Reggae",
  "Highlife", "Juju", "Fuji", "Rock", "Pop", "Traditional", "Contemporary",
];

const instrumentsList = [
  "Piano/Keyboard", "Guitar", "Bass", "Drums", "Saxophone", "Trumpet",
  "Violin", "Cello", "Flute", "Talking Drum", "Shekere", "Voice",
];

const availabilityOptions = ["Weekdays", "Weekends", "Evenings", "Mornings", "Full-time", "Part-time"];

const emptySocialLinks = {
  instagram: "", twitter: "", facebook: "", youtube: "", tiktok: "", spotify: "", soundcloud: "", apple_music: "",
};

export default function TalentOnboarding() {
  const router = useRouter();
  const [step, setStep] = useState(1);
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [form, setForm] = useState({
    name: "",
    bio: "",
    location: "",
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
      router.push("/jobs");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsSubmitting(false);
    }
  }

  const totalSteps = 4;

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background relative py-12 px-4 overflow-hidden">
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_70%_50%_at_50%_0%,var(--accent),transparent)] opacity-60" />
      <div className="relative max-w-2xl mx-auto">
        <div className="mb-8">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-foreground">Step {step} of {totalSteps}</span>
            <span className="text-sm text-muted-foreground">{Math.round((step / totalSteps) * 100)}% complete</span>
          </div>
          <div className="h-2 bg-muted rounded-full overflow-hidden">
            <motion.div
              className="h-full bg-primary rounded-full"
              animate={{ width: `${(step / totalSteps) * 100}%` }}
              transition={{ duration: 0.3 }}
            />
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
                    <div className="grid gap-3">
                      {Object.keys(emptySocialLinks).map((platform) => (
                        <div key={platform}>
                          <Label className="text-sm text-muted-foreground capitalize">{platform.replace("_", " ")}</Label>
                          <Input
                            placeholder={`https://${platform}.com/yourprofile`}
                            value={form.social_links[platform]}
                            onChange={(e) => setSocialLink(platform, e.target.value)}
                            className="mt-1"
                          />
                        </div>
                      ))}
                    </div>
                  </div>
                  <div className="flex items-center gap-4 p-4 bg-muted/50 rounded-xl border border-border">
                    <div className="w-14 h-14 rounded-full bg-primary flex items-center justify-center">
                      <Disc3 className="w-6 h-6 text-primary-foreground" />
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

        <div className="flex justify-between mt-8">
          <Button variant="outline" onClick={() => setStep((s) => s - 1)} disabled={step === 1} className="gap-2">
            <ArrowLeft className="w-4 h-4" />
            Back
          </Button>

          {step < totalSteps ? (
            <Button onClick={() => setStep((s) => s + 1)} className="gap-2">
              Continue
              <ArrowRight className="w-4 h-4" />
            </Button>
          ) : (
            <Button onClick={handleSubmit} disabled={isSubmitting} className="gap-2">
              {isSubmitting ? <Loader2 className="w-4 h-4 animate-spin" /> : <>Complete Profile <Check className="w-4 h-4" /></>}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
