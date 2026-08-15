"use client";

import { useState, useEffect } from "react";
import { useParams, useSearchParams, useRouter } from "next/navigation";
import Link from "next/link";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useMilestones } from "@/hooks/use-milestones";
import { Button } from "@/components/ui/button";
import StatusBadge from "@/components/ui/status-badge";
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
import MilestoneList from "@/components/milestones/MilestoneList";
import CreateMilestonesModal from "@/components/milestones/CreateMilestonesModal";
import DisputeModal from "@/components/disputes/DisputeModal";
import ReviewFormModal from "@/components/reviews/ReviewFormModal";
import { formatMoney } from "@/lib/utils";
import { ArrowLeft, Ban, Loader2, MessageCircle, Plus, ShieldAlert } from "lucide-react";
import { toast } from "sonner";

export default function ContractDetailPage() {
  const { id } = useParams();
  const searchParams = useSearchParams();
  const router = useRouter();
  const { user } = useCurrentUser();
  const queryClient = useQueryClient();
  const [isCompleting, setIsCompleting] = useState(false);
  const [isEnding, setIsEnding] = useState(false);
  // Captured once at mount, independent of `searchParams` — the page's own
  // loading gate below delays CreateMilestonesModal's actual mount until
  // after the contract/user data resolves, by which point the cleanup
  // effect may have already stripped the query param. Deriving this from
  // searchParams directly would race that and lose the flag.
  const [shouldAutoProposeMilestone] = useState(() => searchParams.get("propose") === "1");

  // Consume the ?propose=1 flag once so a refresh doesn't keep reopening
  // the modal — the modal itself keeps its own open state from here on.
  useEffect(() => {
    if (shouldAutoProposeMilestone) {
      router.replace(`/contracts/${id}`);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const { data: contracts, isLoading } = useQuery({
    queryKey: ["contracts", "detail", id],
    queryFn: () => apiGet(`/contracts?id=${id}`),
    enabled: !!id,
  });
  const contract = Array.isArray(contracts) ? contracts[0] : contracts;

  const { milestones, propose, accept, reject, withdraw, counter, fund, release, cancel, requestRelease, refresh } = useMilestones(id);

  async function handleComplete() {
    setIsCompleting(true);
    try {
      await apiPost("/contracts/complete", { contract_id: id });
      toast.success("Contract marked completed.");
      queryClient.invalidateQueries({ queryKey: ["contracts", "detail", id] });
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsCompleting(false);
    }
  }

  async function handleEndContract() {
    setIsEnding(true);
    try {
      const result = await apiPost("/contracts/end", { contract_id: id });
      toast.success(result.status === "disputed" ? "Contract ended — a dispute has been opened for the funded milestone(s)." : "Contract ended.");
      queryClient.invalidateQueries({ queryKey: ["contracts", "detail", id] });
      queryClient.invalidateQueries({ queryKey: ["milestones", id] });
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsEnding(false);
    }
  }

  if (isLoading || !user) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  if (!contract) {
    return <div className="min-h-screen bg-background flex items-center justify-center text-muted-foreground">Contract not found.</div>;
  }

  const role = user.id === contract.client_id ? "client" : "musician";
  const backHref = role === "client" ? "/dashboard/client" : "/profile/contracts";
  const counterpartId = role === "client" ? contract.musician_id : contract.client_id;

  return (
    <div className="min-h-screen bg-background py-12 px-4">
      <div className="max-w-3xl mx-auto space-y-6">
        <Link href={backHref} className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
          <ArrowLeft className="w-4 h-4" />
          {role === "client" ? "Back to dashboard" : "Back to contracts"}
        </Link>

        <div className="bg-card rounded-2xl border border-border p-6">
          <div className="flex items-start justify-between gap-4 flex-wrap">
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-xl font-bold text-foreground">{contract.title || "Contract"}</h1>
                <StatusBadge status={contract.status} />
              </div>
              <p className="text-muted-foreground text-sm mt-1">
                {contract.description || (contract.source === "direct_hire" ? "Direct hire" : "Job-sourced contract")}
              </p>
            </div>
            <div className="text-right">
              <p className="text-2xl font-bold text-foreground tabular-nums">{formatMoney(contract.price)}</p>
              <p className="text-xs text-muted-foreground">Total contract value</p>
            </div>
          </div>

          <div className="flex flex-wrap gap-2 mt-6">
            <Link href={`/messages?with=${counterpartId}&contract=${contract.id}`}>
              <Button size="sm" variant="outline" className="gap-1.5">
                <MessageCircle className="w-3.5 h-3.5" />
                Message
              </Button>
            </Link>
            {role === "client" && contract.status === "active" && (
              <Button size="sm" disabled={isCompleting} onClick={handleComplete} className="gap-1.5">
                {isCompleting && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
                Mark complete
              </Button>
            )}
            {contract.status === "active" && (
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button size="sm" variant="outline" disabled={isEnding} className="gap-1.5 text-muted-foreground hover:text-destructive">
                    <Ban className="w-3.5 h-3.5" />
                    End contract
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>End this contract?</AlertDialogTitle>
                    <AlertDialogDescription>
                      The other party will be notified. If any milestone is currently funded, this opens a dispute instead of
                      ending it outright — a moderator will decide how that money gets settled. Any milestone that&apos;s
                      only been accepted (not yet funded) is cancelled with no dispute.
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>Never mind</AlertDialogCancel>
                    <AlertDialogAction onClick={handleEndContract} className="gap-1.5">
                      End contract
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            )}
            {contract.status === "completed" && (
              <ReviewFormModal
                contractId={contract.id}
                subjectLabel={role === "client" ? "the Talent" : "the client"}
                trigger={<Button size="sm" variant="outline">Leave a review</Button>}
              />
            )}
            <DisputeModal
              contractId={contract.id}
              trigger={
                <Button size="sm" variant="outline" className="gap-1.5">
                  <ShieldAlert className="w-3.5 h-3.5" />
                  Open dispute
                </Button>
              }
            />
          </div>
        </div>

        {contract.escrow_reference ? (
          // Legacy contract from before job hires stopped charging the full
          // budget upfront — the whole amount is already sitting in one
          // escrow agreement that can only release all-or-nothing, so
          // milestones (which need to release incrementally) don't apply.
          <div className="bg-card rounded-2xl border border-border p-6">
            <h2 className="font-semibold text-foreground mb-2">Payment</h2>
            <p className="text-sm text-muted-foreground">
              {role === "client"
                ? `The full contract value (${formatMoney(contract.price)}) was already paid when you hired for this gig — it's held in escrow. Mark the contract complete once the work is done to release payment to the talent.`
                : `The client already paid the full contract value (${formatMoney(contract.price)}) when they hired you — it's held in escrow and will be released once they mark the contract complete.`}
            </p>
          </div>
        ) : (
          <div className="bg-card rounded-2xl border border-border p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-semibold text-foreground">Milestones & escrow</h2>
              {role === "client" && (
                <CreateMilestonesModal
                  trigger={
                    <Button size="sm" variant="outline" className="gap-1.5">
                      <Plus className="w-3.5 h-3.5" />
                      Propose milestone
                    </Button>
                  }
                  onCreate={propose}
                  defaultOpen={shouldAutoProposeMilestone}
                />
              )}
            </div>
            <MilestoneList
              milestones={milestones}
              role={role}
              currentUserId={user.id}
              onAccept={accept}
              onReject={reject}
              onWithdraw={withdraw}
              onCounter={counter}
              onFund={fund}
              onRelease={release}
              onCancel={cancel}
              onRequestRelease={requestRelease}
              onRefresh={refresh}
            />
            <p className="text-xs text-muted-foreground mt-4">
              {role === "client"
                ? "Propose a milestone for the talent to accept, reject, or counter. Once accepted, fund escrow and release it as work is completed."
                : "The client proposes each milestone. You can accept, reject, or counter their offer."}
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
