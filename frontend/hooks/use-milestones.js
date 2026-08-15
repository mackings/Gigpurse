"use client";

import { useMemo } from "react";
import { useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "@/lib/api";

export function useMilestones(contractId) {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ["milestones", contractId],
    queryFn: () => apiGet(`/milestones?contract_id=${contractId}`),
    enabled: !!contractId,
  });

  function invalidate() {
    queryClient.invalidateQueries({ queryKey: ["milestones", contractId] });
    queryClient.invalidateQueries({ queryKey: ["wallet"] });
    queryClient.invalidateQueries({ queryKey: ["wallet-transactions"] });
  }

  async function propose(items) {
    const created = await apiPost("/milestones", { contract_id: contractId, milestones: items });
    invalidate();
    return created;
  }

  async function accept(milestone) {
    const m = await apiPost("/milestones/accept", { contract_id: contractId, milestone_id: milestone.id });
    invalidate();
    return m;
  }

  async function reject(milestone) {
    const m = await apiPost("/milestones/reject", { contract_id: contractId, milestone_id: milestone.id });
    invalidate();
    return m;
  }

  async function withdraw(milestone) {
    const res = await apiPost("/milestones/withdraw", { contract_id: contractId, milestone_id: milestone.id });
    invalidate();
    return res;
  }

  async function counter(milestone, terms) {
    const m = await apiPost("/milestones/counter", { contract_id: contractId, milestone_id: milestone.id, ...terms });
    invalidate();
    return m;
  }

  async function fund(milestone) {
    const m = await apiPost("/milestones/fund", { contract_id: contractId, milestone_id: milestone.id });
    invalidate();
    return m;
  }

  async function release(milestone) {
    const m = await apiPost("/milestones/release", { contract_id: contractId, milestone_id: milestone.id });
    invalidate();
    return m;
  }

  async function cancel(milestone) {
    const m = await apiPost("/milestones/cancel", { contract_id: contractId, milestone_id: milestone.id });
    invalidate();
    return m;
  }

  async function requestRelease(milestone) {
    const res = await apiPost("/milestones/request-release", { contract_id: contractId, milestone_id: milestone.id });
    invalidate();
    return res;
  }

  return {
    milestones: query.data || [],
    isLoading: query.isLoading,
    propose,
    accept,
    reject,
    withdraw,
    counter,
    fund,
    release,
    cancel,
    requestRelease,
    // Exposed so a caller can re-sync after something happens outside this
    // hook's own mutations — e.g. a payment popup closing once PayPetal
    // confirms funding, which this hook has no way to know about on its own.
    refresh: invalidate,
  };
}

// A chat conversation is per person, but milestones are scoped per contract
// — the same two people can rack up more than one contract over time (a
// direct booking, then later a separate job hire, etc). Fetching only
// "the" contract's milestones made every earlier contract's milestones
// disappear from the shared thread the moment a newer contract existed —
// this merges milestones across every contract between the two of them, so
// the chat's milestone panel shows the full history rather than just
// whichever contract happened to resolve as "the" one.
export function useMilestonesForContracts(contractIds) {
  const queryClient = useQueryClient();
  const ids = contractIds || [];

  const results = useQueries({
    queries: ids.map((contractId) => ({
      queryKey: ["milestones", contractId],
      queryFn: () => apiGet(`/milestones?contract_id=${contractId}`),
      enabled: !!contractId,
    })),
  });

  const milestones = useMemo(() => {
    const merged = results.flatMap((r) => r.data || []);
    return merged.sort((a, b) => new Date(a.created_at) - new Date(b.created_at));
  }, [results]);

  function invalidate(contractId) {
    queryClient.invalidateQueries({ queryKey: ["milestones", contractId] });
    queryClient.invalidateQueries({ queryKey: ["wallet"] });
    queryClient.invalidateQueries({ queryKey: ["wallet-transactions"] });
  }

  // Each action takes the milestone itself (not just its id) so it can be
  // routed against its own contract_id — required since the backend
  // verifies a milestone belongs to the contract_id it's given.
  async function propose(contractId, items) {
    const created = await apiPost("/milestones", { contract_id: contractId, milestones: items });
    invalidate(contractId);
    return created;
  }

  async function accept(milestone) {
    const m = await apiPost("/milestones/accept", { contract_id: milestone.contract_id, milestone_id: milestone.id });
    invalidate(milestone.contract_id);
    return m;
  }

  async function reject(milestone) {
    const m = await apiPost("/milestones/reject", { contract_id: milestone.contract_id, milestone_id: milestone.id });
    invalidate(milestone.contract_id);
    return m;
  }

  async function withdraw(milestone) {
    const res = await apiPost("/milestones/withdraw", { contract_id: milestone.contract_id, milestone_id: milestone.id });
    invalidate(milestone.contract_id);
    return res;
  }

  async function counter(milestone, terms) {
    const m = await apiPost("/milestones/counter", { contract_id: milestone.contract_id, milestone_id: milestone.id, ...terms });
    invalidate(milestone.contract_id);
    return m;
  }

  async function fund(milestone) {
    const m = await apiPost("/milestones/fund", { contract_id: milestone.contract_id, milestone_id: milestone.id });
    invalidate(milestone.contract_id);
    return m;
  }

  async function release(milestone) {
    const m = await apiPost("/milestones/release", { contract_id: milestone.contract_id, milestone_id: milestone.id });
    invalidate(milestone.contract_id);
    return m;
  }

  async function cancel(milestone) {
    const m = await apiPost("/milestones/cancel", { contract_id: milestone.contract_id, milestone_id: milestone.id });
    invalidate(milestone.contract_id);
    return m;
  }

  async function requestRelease(milestone) {
    const res = await apiPost("/milestones/request-release", { contract_id: milestone.contract_id, milestone_id: milestone.id });
    invalidate(milestone.contract_id);
    return res;
  }

  return {
    milestones,
    isLoading: results.some((r) => r.isLoading),
    propose,
    accept,
    reject,
    withdraw,
    counter,
    fund,
    release,
    cancel,
    requestRelease,
    // Takes the contractId to re-sync, since this hook spans several.
    refresh: invalidate,
  };
}
