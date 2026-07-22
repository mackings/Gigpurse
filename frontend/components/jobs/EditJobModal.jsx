"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import CurrencyInput from "@/components/ui/currency-input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogTrigger,
} from "@/components/ui/dialog";
import { useQueryClient } from "@tanstack/react-query";
import { apiPut } from "@/lib/api";
import { JOB_DURATION_LABELS, JOB_EXPERIENCE_LABELS, JOB_PROJECT_TYPE_LABELS } from "@/lib/utils";
import { Loader2, X } from "lucide-react";
import { toast } from "sonner";

export default function EditJobModal({ job, trigger, open, onOpenChange }) {
  const queryClient = useQueryClient();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [skillInput, setSkillInput] = useState("");
  const [form, setForm] = useState({
    title: job.title || "",
    description: job.description || "",
    instrument: job.instrument || "",
    genre: job.genre || "",
    location: job.location || "",
    budget: String(job.budget || ""),
    experience_level: job.experience_level || "intermediate",
    duration: job.duration || "less_than_1_week",
    project_type: job.project_type || "one_time",
    skills: job.skills || [],
  });

  function addSkill() {
    const s = skillInput.trim();
    if (s && !form.skills.includes(s)) setForm({ ...form, skills: [...form.skills, s] });
    setSkillInput("");
  }

  function removeSkill(s) {
    setForm({ ...form, skills: form.skills.filter((x) => x !== s) });
  }

  async function handleSubmit(e) {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      await apiPut("/jobs", { job_id: job.id, ...form, budget: parseFloat(form.budget) || 0 });
      toast.success("Gig updated.");
      queryClient.invalidateQueries({ queryKey: ["client-jobs"] });
      queryClient.invalidateQueries({ queryKey: ["job", job.id] });
      onOpenChange(false);
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {trigger && <DialogTrigger asChild>{trigger}</DialogTrigger>}
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Edit gig</DialogTitle>
          <DialogDescription>Applicants with a pending application will be notified this gig was updated.</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4 max-h-[70vh] overflow-y-auto pr-1">
          <div>
            <Label htmlFor="edit-title">Title</Label>
            <Input id="edit-title" required value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} className="mt-1.5" />
          </div>
          <div>
            <Label htmlFor="edit-description">Description</Label>
            <Textarea
              id="edit-description"
              required
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              className="mt-1.5 min-h-[100px]"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="edit-instrument">Instrument</Label>
              <Input
                id="edit-instrument"
                required
                value={form.instrument}
                onChange={(e) => setForm({ ...form, instrument: e.target.value })}
                className="mt-1.5"
              />
            </div>
            <div>
              <Label htmlFor="edit-genre">Genre</Label>
              <Input id="edit-genre" required value={form.genre} onChange={(e) => setForm({ ...form, genre: e.target.value })} className="mt-1.5" />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="edit-location">Location</Label>
              <Input
                id="edit-location"
                required
                value={form.location}
                onChange={(e) => setForm({ ...form, location: e.target.value })}
                className="mt-1.5"
              />
            </div>
            <div>
              <Label htmlFor="edit-budget">Budget (₦, in thousands)</Label>
              <CurrencyInput
                id="edit-budget"
                unit="thousands"
                required
                disabled={job.escrow_funded}
                value={form.budget}
                onChange={(v) => setForm({ ...form, budget: v })}
                className="mt-1.5"
              />
              {job.escrow_funded && <p className="text-xs text-muted-foreground mt-1">Locked — escrow is already funded for this amount.</p>}
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
            <Label htmlFor="edit-skills">Skills</Label>
            <div className="flex gap-2 mt-1.5">
              <Input
                id="edit-skills"
                placeholder="Press Enter to add"
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
                  <span key={s} className="flex items-center gap-1 text-xs font-medium bg-muted text-foreground rounded-full pl-2.5 pr-1.5 py-1">
                    {s}
                    <button type="button" onClick={() => removeSkill(s)} className="hover:text-destructive">
                      <X className="w-3 h-3" />
                    </button>
                  </span>
                ))}
              </div>
            )}
          </div>
          <DialogFooter>
            <Button type="submit" disabled={isSubmitting} className="gap-1.5">
              {isSubmitting && <Loader2 className="w-4 h-4 animate-spin" />}
              Save changes
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
