"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useMilestones } from "@/hooks/use-milestones";
import { Button } from "@/components/ui/button";
import StatusBadge from "@/components/ui/status-badge";
import MilestoneList from "@/components/milestones/MilestoneList";
import CreateMilestonesModal from "@/components/milestones/CreateMilestonesModal";
import DisputeModal from "@/components/disputes/DisputeModal";
import ReviewFormModal from "@/components/reviews/ReviewFormModal";
import { formatMoney } from "@/lib/utils";
import { ArrowLeft, Loader2, MessageCircle, Plus, ShieldAlert } from "lucide-react";
import { toast } from "sonner";

export default function ContractDetailPage() {
  const { id } = useParams();
  const { user } = useCurrentUser();
  const queryClient = useQueryClient();

  const { data: contracts, isLoading } = useQuery({
    queryKey: ["contracts", "detail", id],
    queryFn: () => apiGet(`/contracts?id=${id}`),
    enabled: !!id,
  });
  const contract = Array.isArray(contracts) ? contracts[0] : contracts;

  const { milestones, propose, accept, reject, counter, fund, release } = useMilestones(id);

  async function handleComplete() {
    try {
      await apiPost("/contracts/complete", { contract_id: id });
      toast.success("Contract marked completed.");
      queryClient.invalidateQueries({ queryKey: ["contracts", "detail", id] });
    } catch (err) {
      toast.error(err.message);
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
              <Button size="sm" onClick={handleComplete}>Mark complete</Button>
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

        <div className="bg-card rounded-2xl border border-border p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="font-semibold text-foreground">Milestones & escrow</h2>
            <CreateMilestonesModal
              trigger={
                <Button size="sm" variant="outline" className="gap-1.5">
                  <Plus className="w-3.5 h-3.5" />
                  Propose milestone
                </Button>
              }
              onCreate={propose}
            />
          </div>
          <MilestoneList
            milestones={milestones}
            role={role}
            currentUserId={user.id}
            onAccept={accept}
            onReject={reject}
            onCounter={counter}
            onFund={fund}
            onRelease={release}
          />
          <p className="text-xs text-muted-foreground mt-4">
            Either party can propose a milestone. The other party accepts or rejects it, then the client funds and
            releases escrow as work is completed.
          </p>
        </div>
      </div>
    </div>
  );
}
