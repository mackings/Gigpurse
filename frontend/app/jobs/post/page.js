"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import CurrencyInput from "@/components/ui/currency-input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { useWallet } from "@/hooks/use-wallet";
import { formatMoney, JOB_DURATION_LABELS, JOB_EXPERIENCE_LABELS, JOB_PROJECT_TYPE_LABELS } from "@/lib/utils";
import { Loader2, X, ShieldCheck, Wallet as WalletIcon } from "lucide-react";
import { toast } from "sonner";
import { apiPost } from "@/lib/api";

export default function PostJob() {
  const router = useRouter();
  const { wallet, isLoading: walletLoading, deposit } = useWallet();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isFunding, setIsFunding] = useState(false);
  const [isDepositing, setIsDepositing] = useState(false);
  const [topUpAmount, setTopUpAmount] = useState("");
  const [postedJob, setPostedJob] = useState(null);
  const [skillInput, setSkillInput] = useState("");
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
      toast.success("Gig details saved — now fund escrow to publish it.");
      setPostedJob(job);
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleFundEscrow() {
    setIsFunding(true);
    try {
      await apiPost("/jobs/fund", { job_id: postedJob.id });
      toast.success("Escrow funded — your gig is live!");
      router.push("/dashboard/client");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsFunding(false);
    }
  }

  async function handleTopUp(e) {
    e.preventDefault();
    setIsDepositing(true);
    try {
      await deposit(parseFloat(topUpAmount) || 0);
      toast.success("Funds added.");
      setTopUpAmount("");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsDepositing(false);
    }
  }

  if (postedJob) {
    const balance = wallet?.balance ?? 0;
    const needed = postedJob.budget;
    const canFund = balance >= needed;

    return (
      <div className="min-h-screen bg-background py-12 px-4">
        <div className="max-w-xl mx-auto">
          <Card>
            <CardHeader>
              <CardTitle className="text-2xl">Fund escrow to publish</CardTitle>
              <CardDescription>
                Gigs only become visible to talent once their budget is secured in escrow — this guarantees payment for
                whoever you hire.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              <div className="rounded-xl border border-border p-4">
                <p className="font-semibold text-foreground">{postedJob.title}</p>
                <p className="text-sm text-muted-foreground mt-1">{postedJob.location}</p>
                <p className="text-lg font-bold text-foreground mt-2">{formatMoney(needed)}</p>
              </div>

              <div className="flex items-center justify-between rounded-xl bg-muted/40 p-4">
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <WalletIcon className="w-4 h-4" />
                  Wallet balance
                </div>
                <span className="font-semibold text-foreground tabular-nums">
                  {walletLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : formatMoney(balance, { decimals: 2 })}
                </span>
              </div>

              {!canFund && !walletLoading && (
                <form onSubmit={handleTopUp} className="space-y-2">
                  <Label className="text-xs text-muted-foreground">
                    Insufficient balance — top up at least {formatMoney(needed - balance)} to fund this gig
                  </Label>
                  <div className="flex gap-2">
                    <CurrencyInput
                      required
                      placeholder="Amount (₦)"
                      value={topUpAmount}
                      onChange={setTopUpAmount}
                      className="flex-1"
                    />
                    <Button type="submit" variant="outline" disabled={isDepositing}>
                      {isDepositing ? <Loader2 className="w-4 h-4 animate-spin" /> : "Add funds"}
                    </Button>
                  </div>
                </form>
              )}

              <Button className="w-full gap-2" disabled={!canFund || isFunding} onClick={handleFundEscrow}>
                {isFunding ? <Loader2 className="w-4 h-4 animate-spin" /> : <ShieldCheck className="w-4 h-4" />}
                Fund escrow &amp; publish gig
              </Button>
              <p className="text-xs text-muted-foreground text-center">
                Your gig is saved as a draft until funded — you can come back and fund it later from your dashboard.
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
                  <Label htmlFor="budget">Budget (₦, in thousands)</Label>
                  <CurrencyInput
                    id="budget"
                    unit="thousands"
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
                {isSubmitting ? <Loader2 className="w-4 h-4 animate-spin" /> : "Continue to fund escrow"}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
