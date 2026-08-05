"use client";

import { useCurrentUser } from "@/hooks/use-current-user";
import { useWallet } from "@/hooks/use-wallet";
import WalletCard from "@/components/wallet/WalletCard";
import TransactionList from "@/components/wallet/TransactionList";
import { formatMoney } from "@/lib/utils";
import { Loader2 } from "lucide-react";

export default function WalletPage() {
  const { user } = useCurrentUser();
  const { wallet, transactions, isLoading } = useWallet();

  if (isLoading || !user) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background py-12 px-4">
      <div className="max-w-3xl mx-auto space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground tracking-tight">Wallet</h1>
          <p className="text-muted-foreground">Your payout account and payment history.</p>
        </div>

        <WalletCard payoutAccount={wallet?.payout_account} />

        <div className="grid sm:grid-cols-2 gap-4">
          <div className="group relative overflow-hidden bg-card rounded-2xl border border-border p-5 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-emerald-500/20 hover:-translate-y-0.5">
            <div className="absolute -top-6 -right-6 w-24 h-24 rounded-full bg-emerald-500 opacity-[0.08] blur-2xl transition-opacity duration-300 group-hover:opacity-[0.16]" />
            <p className="relative text-sm text-muted-foreground">Total earned</p>
            <p className="relative text-2xl font-bold text-foreground tabular-nums">{formatMoney(wallet?.total_earned || 0, { decimals: 2 })}</p>
          </div>
          <div className="group relative overflow-hidden bg-card rounded-2xl border border-border p-5 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-rose-500/20 hover:-translate-y-0.5">
            <div className="absolute -top-6 -right-6 w-24 h-24 rounded-full bg-rose-500 opacity-[0.08] blur-2xl transition-opacity duration-300 group-hover:opacity-[0.16]" />
            <p className="relative text-sm text-muted-foreground">Total spent</p>
            <p className="relative text-2xl font-bold text-foreground tabular-nums">{formatMoney(wallet?.total_spent || 0, { decimals: 2 })}</p>
          </div>
        </div>

        <div className="bg-card rounded-2xl border border-border p-6">
          <h2 className="font-semibold text-foreground mb-4">Transaction history</h2>
          <TransactionList transactions={transactions} />
        </div>
      </div>
    </div>
  );
}
