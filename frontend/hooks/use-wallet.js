"use client";

import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";

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
