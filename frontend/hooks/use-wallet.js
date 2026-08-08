"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "@/lib/api";

export function useWallet() {
  const walletQuery = useQuery({
    queryKey: ["wallet"],
    queryFn: () => apiGet("/wallet"),
  });

  const transactionsQuery = useQuery({
    queryKey: ["wallet-transactions"],
    queryFn: () => apiGet("/wallet/transactions"),
  });

  return {
    wallet: walletQuery.data,
    transactions: transactionsQuery.data || [],
    isLoading: walletQuery.isLoading,
  };
}

// Client-only: every payment they currently have held in escrow (a job hire
// or a funded milestone that hasn't paid out or been refunded yet), each
// refundable individually. There's no reloadable balance with PayPetal —
// this is the closest thing a client has to "money in escrow."
export function useEscrowHoldings() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ["wallet-escrow"],
    queryFn: () => apiGet("/wallet/escrow"),
  });

  async function requestRefund(referenceId) {
    await apiPost("/wallet/escrow/refund", { reference_id: referenceId });
    queryClient.invalidateQueries({ queryKey: ["wallet-escrow"] });
    queryClient.invalidateQueries({ queryKey: ["wallet-transactions"] });
    queryClient.invalidateQueries({ queryKey: ["wallet"] });
  }

  return {
    holdings: query.data || [],
    isLoading: query.isLoading,
    requestRefund,
  };
}
