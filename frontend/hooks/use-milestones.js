"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
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

  async function accept(milestoneId) {
    const m = await apiPost("/milestones/accept", { contract_id: contractId, milestone_id: milestoneId });
    invalidate();
    return m;
  }

  async function reject(milestoneId) {
    const m = await apiPost("/milestones/reject", { contract_id: contractId, milestone_id: milestoneId });
    invalidate();
    return m;
  }

  async function withdraw(milestoneId) {
    const res = await apiPost("/milestones/withdraw", { contract_id: contractId, milestone_id: milestoneId });
    invalidate();
    return res;
  }

  async function counter(milestoneId, terms) {
    const m = await apiPost("/milestones/counter", { contract_id: contractId, milestone_id: milestoneId, ...terms });
    invalidate();
    return m;
  }

  async function fund(milestoneId) {
    const m = await apiPost("/milestones/fund", { contract_id: contractId, milestone_id: milestoneId });
    invalidate();
    return m;
  }

  async function release(milestoneId) {
    const m = await apiPost("/milestones/release", { contract_id: contractId, milestone_id: milestoneId });
    invalidate();
    return m;
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
  };
}
