"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "@/lib/api";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import StatusBadge from "@/components/ui/status-badge";
import MediaThumb from "@/components/portfolio/MediaThumb";
import EditJobModal from "@/components/jobs/EditJobModal";
import {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogAction,
  AlertDialogCancel,
} from "@/components/ui/alert-dialog";
import { formatMoney } from "@/lib/utils";
import { MapPin, Star, Pencil, XCircle, Loader2, Check } from "lucide-react";
import { toast } from "sonner";

function ApplicantRow({ app, jobStatus, onAccept, isAccepting }) {
  const a = app.applicant;
  return (
    <div className="p-4 rounded-xl border border-border bg-card space-y-3">
      <div className="flex items-start gap-3">
        <div className="w-9 h-9 rounded-full bg-primary flex items-center justify-center text-primary-foreground text-sm font-semibold shrink-0">
          {(a?.name || "?").charAt(0).toUpperCase()}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center justify-between gap-2">
            <p className="font-medium text-foreground truncate">{a?.name || "Musician"}</p>
            <StatusBadge status={app.status} />
          </div>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5 mt-0.5 text-xs text-muted-foreground">
            {a?.review_count > 0 && (
              <span className="flex items-center gap-1">
                <Star className="w-3 h-3 text-amber-500 fill-amber-500" />
                {a.rating.toFixed(1)} ({a.review_count})
              </span>
            )}
            {a?.location && (
              <span className="flex items-center gap-1">
                <MapPin className="w-3 h-3" />
                {a.location}
              </span>
            )}
            {a?.genres?.length > 0 && <span>{a.genres.join(", ")}</span>}
            {a?.instruments?.length > 0 && <span>{a.instruments.join(", ")}</span>}
          </div>
        </div>
      </div>

      <p className="text-sm text-foreground">{app.proposal}</p>

      {app.portfolio_items?.length > 0 && (
        <div className="flex gap-1.5 overflow-x-auto pb-1">
          {app.portfolio_items.map((item, i) => (
            <div key={item.id || i} className="w-14 h-14 rounded-lg overflow-hidden border border-border shrink-0" title={item.title}>
              <MediaThumb item={item} className="rounded-none" />
            </div>
          ))}
        </div>
      )}

      <div className="flex items-center justify-between pt-1">
        <span className="text-sm font-semibold text-foreground">Bid: {formatMoney(app.price_bid)}</span>
        {app.status === "pending" && jobStatus === "open" && (
          <Button size="sm" disabled={isAccepting} onClick={() => onAccept(app.id)} className="gap-1.5">
            {isAccepting ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Check className="w-3.5 h-3.5" />}
            Accept
          </Button>
        )}
      </div>
    </div>
  );
}

export default function ApplicantsPanel({ job, open, onOpenChange }) {
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);
  const [acceptingId, setAcceptingId] = useState(null);
  const [isClosing, setIsClosing] = useState(false);

  const { data: applications, isLoading } = useQuery({
    queryKey: ["job-applications", job.id],
    queryFn: () => apiGet(`/jobs/applications?job_id=${job.id}`),
    enabled: open,
  });

  async function accept(applicationId) {
    setAcceptingId(applicationId);
    try {
      await apiPost("/jobs/applications/accept", { application_id: applicationId });
      toast.success("Application accepted!");
      queryClient.invalidateQueries({ queryKey: ["job-applications", job.id] });
      queryClient.invalidateQueries({ queryKey: ["client-jobs"] });
      queryClient.invalidateQueries({ queryKey: ["contracts"] });
    } catch (err) {
      toast.error(err.message);
    } finally {
      setAcceptingId(null);
    }
  }

  async function closeJob() {
    setIsClosing(true);
    try {
      await apiPost("/jobs/close", { job_id: job.id });
      toast.success("Gig closed — it's no longer accepting applications.");
      queryClient.invalidateQueries({ queryKey: ["client-jobs"] });
      onOpenChange(false);
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsClosing(false);
    }
  }

  return (
    <>
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent>
          <SheetHeader>
            <SheetTitle className="truncate">{job.title}</SheetTitle>
            <SheetDescription className="sr-only">Job details and applicants</SheetDescription>
          </SheetHeader>
          <div className="overflow-y-auto flex-1 p-4 sm:p-5 space-y-5">
            <div className="rounded-xl border border-border p-4 space-y-2">
              <div className="flex items-center justify-between gap-2">
                <StatusBadge status={job.status} />
                <span className="font-semibold text-foreground">{formatMoney(job.budget)}</span>
              </div>
              <p className="text-sm text-muted-foreground flex items-center gap-1">
                <MapPin className="w-3.5 h-3.5" />
                {job.location}
              </p>
              <p className="text-sm text-foreground line-clamp-3">{job.description}</p>
              <div className="flex gap-2 pt-1">
                <Button size="sm" variant="outline" className="gap-1.5" onClick={() => setEditOpen(true)}>
                  <Pencil className="w-3.5 h-3.5" />
                  Edit gig
                </Button>
                {job.status === "open" && (
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button size="sm" variant="outline" className="gap-1.5 text-destructive hover:text-destructive">
                        <XCircle className="w-3.5 h-3.5" />
                        Close job
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Close this gig?</AlertDialogTitle>
                        <AlertDialogDescription>
                          It&apos;ll stop accepting applications immediately. Anyone with a pending application will be notified.
                          {job.escrow_funded && " Since escrow was already funded, that amount is refunded back to your wallet."}
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction onClick={closeJob} disabled={isClosing} className="gap-1.5">
                          {isClosing && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
                          Close job
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                )}
              </div>
            </div>

            <div>
              <h3 className="font-semibold text-foreground mb-3">
                Applicants {applications?.length > 0 && `(${applications.length})`}
              </h3>
              {isLoading ? (
                <div className="flex justify-center py-8">
                  <Loader2 className="w-5 h-5 animate-spin text-primary" />
                </div>
              ) : applications?.length ? (
                <div className="space-y-3">
                  {applications.map((app) => (
                    <ApplicantRow key={app.id} app={app} jobStatus={job.status} onAccept={accept} isAccepting={acceptingId === app.id} />
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No applications yet.</p>
              )}
            </div>
          </div>
        </SheetContent>
      </Sheet>

      <EditJobModal job={job} open={editOpen} onOpenChange={setEditOpen} />
    </>
  );
}
