"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "@/lib/api";

export function useWallet() {
  const queryClient = useQueryClient();

  const walletQuery = useQuery({
    queryKey: ["wallet"],
    queryFn: () => apiGet("/wallet"),
  });

  const transactionsQuery = useQuery({
    queryKey: ["wallet-transactions"],
    queryFn: () => apiGet("/wallet/transactions"),
  });

  function invalidate() {
    queryClient.invalidateQueries({ queryKey: ["wallet"] });
    queryClient.invalidateQueries({ queryKey: ["wallet-transactions"] });
  }

  async function deposit(amount) {
    const wallet = await apiPost("/wallet/deposit", { amount });
    invalidate();
    return wallet;
  }

  async function withdraw(amount) {
    const wallet = await apiPost("/wallet/withdraw", { amount });
    invalidate();
    return wallet;
  }

  return {
    wallet: walletQuery.data,
    transactions: transactionsQuery.data || [],
    isLoading: walletQuery.isLoading,
    deposit,
    withdraw,
  };
}
