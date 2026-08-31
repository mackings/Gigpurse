"use client";

import { useState, useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import CurrencyInput from "@/components/ui/currency-input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import AddPayoutAccountSheet from "@/components/wallet/AddPayoutAccountSheet";
import { useCurrentUser } from "@/hooks/use-current-user";
import { formatMoney, JOB_DURATION_LABELS, JOB_EXPERIENCE_LABELS, JOB_PROJECT_TYPE_LABELS } from "@/lib/utils";
import { Loader2, X, Megaphone, Landmark, AlertCircle } from "lucide-react";
import { toast } from "sonner";
import { apiPost } from "@/lib/api";

export default function PostJob() {
  const router = useRouter();
  const { user, refetch: refetchUser } = useCurrentUser();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isPublishing, setIsPublishing] = useState(false);
  const [postedJob, setPostedJob] = useState(null);
  const [skillInput, setSkillInput] = useState("");
  const [payoutSheetOpen, setPayoutSheetOpen] = useState(false);
  const hasPayoutAccount = !!user?.payout_account;
  const hasAutoPrompted = useRef(false);

  // Detect the missing payout account up front — before the user fills out
  // the whole form and gets surprised by it on submit — and prompt once per
  // visit. A manual "Add payout account" banner stays visible afterward in
  // case they dismiss the sheet without adding one.
  useEffect(() => {
    if (user && !hasPayoutAccount && !hasAutoPrompted.current) {
      hasAutoPrompted.current = true;
      setPayoutSheetOpen(true);
    }
  }, [user, hasPayoutAccount]);
  const [form, setForm] = useState({
    title: "",
    description: "",
    instrument: "",
    genre: "",
    location: "",
    budget: "",
    experience_level: "intermediate",
    duration: "less_than_1_week",
    project_type: "one_time",
    skills: [],
  });

  function addSkill() {
    const s = skillInput.trim();
    if (s && !form.skills.includes(s)) {
      setForm({ ...form, skills: [...form.skills, s] });
    }
    setSkillInput("");
  }

  function removeSkill(s) {
    setForm({ ...form, skills: form.skills.filter((x) => x !== s) });
  }

  async function handleSubmit(e) {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      const job = await apiPost("/jobs", { ...form, budget: parseFloat(form.budget) || 0 });
      toast.success("Gig details saved — publish it to start receiving applications.");
      setPostedJob(job);
    } catch (err) {
      if (err.code === "payment_required") {
        toast.error("You have a dispute settlement waiting to be paid — check your contracts and fund it before posting a new gig.");
      } else if (err.code === "payout_account_required") {
        toast.error("Add a payout account before posting a gig — it's what any future dispute refund would be paid back to.");
        setPayoutSheetOpen(true);
      } else {
        toast.error(err.message);
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handlePublish() {
    setIsPublishing(true);
    try {
      await apiPost("/jobs/fund", { job_id: postedJob.id });
      toast.success("Your gig is live!");
      router.push("/dashboard/client");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsPublishing(false);
    }
  }

  if (postedJob) {
    return (
      <div className="min-h-screen bg-background py-12 px-4">
        <div className="max-w-xl mx-auto">
          <Card>
            <CardHeader>
              <CardTitle className="text-2xl">Publish your gig</CardTitle>
              <CardDescription>
                No payment is due yet — you&apos;ll pay securely when you accept a specific applicant, and that amount is
                held in escrow until the work is done.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              <div className="rounded-xl border border-border p-4">
                <p className="font-semibold text-foreground">{postedJob.title}</p>
                <p className="text-sm text-muted-foreground mt-1">{postedJob.location}</p>
                <p className="text-lg font-bold text-foreground mt-2">{formatMoney(postedJob.budget)}</p>
              </div>

              <Button className="w-full gap-2" disabled={isPublishing} onClick={handlePublish}>
                {isPublishing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Megaphone className="w-4 h-4" />}
                Publish gig
              </Button>
              <p className="text-xs text-muted-foreground text-center">
                Your gig is saved as a draft until published — you can come back and publish it later from your
                dashboard.
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background py-12 px-4">
      <div className="max-w-xl mx-auto">
        {!hasPayoutAccount && (
          <div className="flex items-center justify-between gap-3 flex-wrap rounded-xl border border-status-warning/30 bg-status-warning/10 px-4 py-3 mb-5">
            <div className="flex items-center gap-2.5 text-sm text-foreground">
              <AlertCircle className="w-4 h-4 text-status-warning shrink-0" />
              Add a payout account before posting — it&apos;s where a dispute refund would be paid back to.
            </div>
            <Button type="button" size="sm" variant="outline" className="gap-1.5 shrink-0" onClick={() => setPayoutSheetOpen(true)}>
              <Landmark className="w-3.5 h-3.5" />
              Add payout account
            </Button>
          </div>
        )}
        <Card>
          <CardHeader>
            <CardTitle className="text-2xl">Post a gig</CardTitle>
            <CardDescription>Tell Talent what you need for your event.</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <Label htmlFor="title">Title</Label>
                <Input
                  id="title"
                  required
                  placeholder="Afrobeats guitarist for wedding reception"
                  value={form.title}
                  onChange={(e) => setForm({ ...form, title: e.target.value })}
                  className="mt-1.5"
                />
              </div>
              <div>
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  required
                  placeholder="Describe the event, date, and what you're looking for..."
                  value={form.description}
                  onChange={(e) => setForm({ ...form, description: e.target.value })}
                  className="mt-1.5 min-h-[120px]"
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="instrument">Instrument</Label>
                  <Input
                    id="instrument"
                    required
                    placeholder="Guitar"
                    value={form.instrument}
                    onChange={(e) => setForm({ ...form, instrument: e.target.value })}
                    className="mt-1.5"
                  />
                </div>
                <div>
                  <Label htmlFor="genre">Genre</Label>
                  <Input
                    id="genre"
                    required
                    placeholder="Afrobeats"
                    value={form.genre}
                    onChange={(e) => setForm({ ...form, genre: e.target.value })}
                    className="mt-1.5"
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="location">Location</Label>
                  <Input
                    id="location"
                    required
                    placeholder="Lagos"
                    value={form.location}
                    onChange={(e) => setForm({ ...form, location: e.target.value })}
                    className="mt-1.5"
                  />
                </div>
                <div>
                  <Label htmlFor="budget">Budget (₦)</Label>
                  <CurrencyInput
                    id="budget"
                    required
                    value={form.budget}
                    onChange={(v) => setForm({ ...form, budget: v })}
                    className="mt-1.5"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label>Experience level</Label>
                  <Select value={form.experience_level} onValueChange={(v) => setForm({ ...form, experience_level: v })}>
                    <SelectTrigger className="mt-1.5">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {Object.entries(JOB_EXPERIENCE_LABELS).map(([v, label]) => (
                        <SelectItem key={v} value={v}>
                          {label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div>
                  <Label>Duration</Label>
                  <Select value={form.duration} onValueChange={(v) => setForm({ ...form, duration: v })}>
                    <SelectTrigger className="mt-1.5">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {Object.entries(JOB_DURATION_LABELS).map(([v, label]) => (
                        <SelectItem key={v} value={v}>
                          {label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div>
                <Label>Project type</Label>
                <Select value={form.project_type} onValueChange={(v) => setForm({ ...form, project_type: v })}>
                  <SelectTrigger className="mt-1.5">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {Object.entries(JOB_PROJECT_TYPE_LABELS).map(([v, label]) => (
                      <SelectItem key={v} value={v}>
                        {label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div>
                <Label htmlFor="skills">Skills (optional)</Label>
                <div className="flex gap-2 mt-1.5">
                  <Input
                    id="skills"
                    placeholder="e.g. Sight-reading — press Enter to add"
                    value={skillInput}
                    onChange={(e) => setSkillInput(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") {
                        e.preventDefault();
                        addSkill();
                      }
                    }}
                  />
                  <Button type="button" variant="outline" onClick={addSkill}>
                    Add
                  </Button>
                </div>
                {form.skills.length > 0 && (
                  <div className="flex flex-wrap gap-1.5 mt-2">
                    {form.skills.map((s) => (
                      <span
                        key={s}
                        className="flex items-center gap-1 text-xs font-medium bg-muted text-foreground rounded-full pl-2.5 pr-1.5 py-1"
                      >
                        {s}
                        <button type="button" onClick={() => removeSkill(s)} className="hover:text-destructive">
                          <X className="w-3 h-3" />
                        </button>
                      </span>
                    ))}
                  </div>
                )}
              </div>

              <Button type="submit" disabled={isSubmitting} className="w-full">
                {isSubmitting ? <Loader2 className="w-4 h-4 animate-spin" /> : "Post gig"}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>

      <AddPayoutAccountSheet open={payoutSheetOpen} onOpenChange={setPayoutSheetOpen} onSaved={refetchUser} />
    </div>
  );
}
