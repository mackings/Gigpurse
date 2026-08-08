"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost, apiDelete } from "@/lib/api";
import { openPaymentPopup } from "@/lib/payment-popup";
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
import { MapPin, Star, Pencil, XCircle, Trash2, Loader2, Check, UserRound, HandCoins, BookmarkPlus, BookmarkMinus } from "lucide-react";
import { toast } from "sonner";

function ApplicantRow({ app, jobStatus, contractId, onAccept, isAccepting, onShortlist, onUnshortlist, isTogglingShortlist }) {
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

      <div className="flex items-center justify-between pt-1 flex-wrap gap-2">
        <span className="text-sm font-semibold text-foreground">Bid: {formatMoney(app.price_bid)}</span>
        <div className="flex items-center gap-2 flex-wrap">
          <Button asChild size="sm" variant="outline" className="gap-1.5">
            <Link href={`/talent/${app.musician_id}`}>
              <UserRound className="w-3.5 h-3.5" />
              View profile
            </Link>
          </Button>
          {app.status === "pending" && jobStatus === "open" && (
            <Button size="sm" variant="outline" disabled={isTogglingShortlist} onClick={() => onShortlist(app.id)} className="gap-1.5">
              {isTogglingShortlist ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <BookmarkPlus className="w-3.5 h-3.5" />}
              Shortlist
            </Button>
          )}
          {app.status === "shortlisted" && jobStatus === "open" && (
            <Button size="sm" variant="outline" disabled={isTogglingShortlist} onClick={() => onUnshortlist(app.id)} className="gap-1.5">
              {isTogglingShortlist ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <BookmarkMinus className="w-3.5 h-3.5" />}
              Remove
            </Button>
          )}
          {(app.status === "pending" || app.status === "shortlisted") && jobStatus === "open" && (
            <Button size="sm" disabled={isAccepting} onClick={() => onAccept(app.id)} className="gap-1.5">
              {isAccepting ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Check className="w-3.5 h-3.5" />}
              Accept
            </Button>
          )}
          {app.status === "accepted" && contractId && (
            <Button asChild size="sm" className="gap-1.5">
              <Link href={`/contracts/${contractId}?propose=1`}>
                <HandCoins className="w-3.5 h-3.5" />
                Propose milestone
              </Link>
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}

export default function ApplicantsPanel({ job, open, onOpenChange }) {
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);
  const [acceptingId, setAcceptingId] = useState(null);
  const [isClosing, setIsClosing] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [togglingShortlistId, setTogglingShortlistId] = useState(null);
  const [tab, setTab] = useState("all");

  const { data: applications, isLoading } = useQuery({
    queryKey: ["job-applications", job.id],
    queryFn: () => apiGet(`/jobs/applications?job_id=${job.id}`),
    enabled: open,
  });

  // The application record itself doesn't carry a contract_id, so once
  // someone's accepted, look their contract up by matching this job +
  // musician against the client's contracts — covers applications accepted
  // before this "propose a milestone" shortcut existed too.
  const { data: contracts } = useQuery({
    queryKey: ["contracts"],
    queryFn: () => apiGet("/contracts"),
    enabled: open && applications?.some((a) => a.status === "accepted"),
  });
  const contractIdByMusician = {};
  for (const c of contracts || []) {
    if (c.job_id === job.id) contractIdByMusician[c.musician_id] = c.id;
  }

  const shortlistedCount = applications?.filter((a) => a.status === "shortlisted").length || 0;
  const visibleApplications = tab === "shortlisted" ? applications?.filter((a) => a.status === "shortlisted") : applications;

  async function accept(applicationId) {
    setAcceptingId(applicationId);
    try {
      const result = await apiPost("/jobs/applications/accept", { application_id: applicationId });
      queryClient.invalidateQueries({ queryKey: ["job-applications", job.id] });
      queryClient.invalidateQueries({ queryKey: ["client-jobs"] });
      // Hiring now means paying — open PayPetal's hosted checkout in a
      // popup instead of navigating away. The contract itself isn't created
      // until that payment is confirmed (see FinalizeHire, triggered by the
      // webhook or the popup's own poll) — once the popup closes, re-sync so
      // the confirmed hire shows up without a manual refresh.
      if (result?.payment_url) {
        openPaymentPopup(result.payment_url, {
          onClose: () => {
            queryClient.invalidateQueries({ queryKey: ["job-applications", job.id] });
            queryClient.invalidateQueries({ queryKey: ["client-jobs"] });
            queryClient.invalidateQueries({ queryKey: ["contracts"] });
          },
        });
        return;
      }
    } catch (err) {
      if (err.code === "payout_account_required") {
        toast.error("This musician hasn't added a payout account yet — they've been notified to add one.");
      } else {
        toast.error(err.message);
      }
    } finally {
      setAcceptingId(null);
    }
  }

  async function shortlist(applicationId) {
    setTogglingShortlistId(applicationId);
    try {
      await apiPost("/jobs/applications/shortlist", { application_id: applicationId });
      queryClient.invalidateQueries({ queryKey: ["job-applications", job.id] });
    } catch (err) {
      toast.error(err.message);
    } finally {
      setTogglingShortlistId(null);
    }
  }

  async function unshortlist(applicationId) {
    setTogglingShortlistId(applicationId);
    try {
      await apiPost("/jobs/applications/unshortlist", { application_id: applicationId });
      queryClient.invalidateQueries({ queryKey: ["job-applications", job.id] });
    } catch (err) {
      toast.error(err.message);
    } finally {
      setTogglingShortlistId(null);
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

  async function deleteJob() {
    setIsDeleting(true);
    try {
      await apiDelete("/jobs", { job_id: job.id });
      toast.success("Draft gig deleted.");
      queryClient.invalidateQueries({ queryKey: ["client-jobs"] });
      onOpenChange(false);
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsDeleting(false);
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
                          It&apos;ll stop accepting applications immediately. Anyone with a pending or shortlisted application will
                          be notified.
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
                {job.status === "pending_funding" && (
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button size="sm" variant="outline" className="gap-1.5 text-destructive hover:text-destructive">
                        <Trash2 className="w-3.5 h-3.5" />
                        Delete
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Delete this draft gig?</AlertDialogTitle>
                        <AlertDialogDescription>
                          It was never published, so this removes it for good — there&apos;s nothing to undo.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction onClick={deleteJob} disabled={isDeleting} className="gap-1.5">
                          {isDeleting && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
                          Delete
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                )}
              </div>
            </div>

            <div>
              <div className="flex items-center justify-between mb-3">
                <h3 className="font-semibold text-foreground">
                  Applicants {applications?.length > 0 && `(${applications.length})`}
                </h3>
                {shortlistedCount > 0 && (
                  <div className="flex items-center gap-1 rounded-full bg-muted p-0.5 text-xs">
                    <button
                      type="button"
                      onClick={() => setTab("all")}
                      className={`px-2.5 py-1 rounded-full font-medium transition-colors ${tab === "all" ? "bg-card text-foreground shadow-sm" : "text-muted-foreground"}`}
                    >
                      All
                    </button>
                    <button
                      type="button"
                      onClick={() => setTab("shortlisted")}
                      className={`px-2.5 py-1 rounded-full font-medium transition-colors ${tab === "shortlisted" ? "bg-card text-foreground shadow-sm" : "text-muted-foreground"}`}
                    >
                      Shortlisted ({shortlistedCount})
                    </button>
                  </div>
                )}
              </div>
              {isLoading ? (
                <div className="flex justify-center py-8">
                  <Loader2 className="w-5 h-5 animate-spin text-primary" />
                </div>
              ) : visibleApplications?.length ? (
                <div className="space-y-3">
                  {visibleApplications.map((app) => (
                    <ApplicantRow
                      key={app.id}
                      app={app}
                      jobStatus={job.status}
                      contractId={contractIdByMusician[app.musician_id]}
                      onAccept={accept}
                      isAccepting={acceptingId === app.id}
                      onShortlist={shortlist}
                      onUnshortlist={unshortlist}
                      isTogglingShortlist={togglingShortlistId === app.id}
                    />
                  ))}
                </div>
              ) : tab === "shortlisted" ? (
                <p className="text-sm text-muted-foreground">No shortlisted applicants yet.</p>
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
