"use client";

import { Briefcase } from "lucide-react";

// The list itself lives in layout.js (shared with /admin/jobs/[id] so it
// stays mounted while browsing between jobs) — this route only ever
// renders when no specific job is selected yet.
export default function AdminJobs() {
  return (
    <div className="hidden lg:flex flex-col items-center justify-center gap-3 h-full min-h-[400px] rounded-xl border border-dashed border-border text-center px-6">
      <div className="w-11 h-11 rounded-lg bg-muted flex items-center justify-center">
        <Briefcase className="w-5 h-5 text-muted-foreground" />
      </div>
      <div>
        <p className="font-medium text-foreground">Select a job</p>
        <p className="text-sm text-muted-foreground mt-1">
          Choose a job from the list to see its applications, contract, and escrow status.
        </p>
      </div>
    </div>
  );
}
