"use client";

import { useCurrentUser } from "@/hooks/use-current-user";
import { useWallet } from "@/hooks/use-wallet";
import WalletCard from "@/components/wallet/WalletCard";
import WithdrawModal from "@/components/wallet/WithdrawModal";
import TransactionList from "@/components/wallet/TransactionList";
import { Button } from "@/components/ui/button";
import { Loader2, ArrowUpRight } from "lucide-react";

export default function WalletPage() {
  const { user } = useCurrentUser();
  const { wallet, transactions, isLoading, deposit, withdraw } = useWallet();

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
          <p className="text-muted-foreground">Manage your balance, escrow, and payment history.</p>
        </div>

        <div className="grid sm:grid-cols-[1fr_auto] gap-4 items-start">
          <WalletCard balance={wallet?.balance} escrowBalance={wallet?.escrow_balance} onDeposit={deposit} />
          <WithdrawModal
            onWithdraw={withdraw}
            trigger={
              <Button variant="outline" className="gap-2 sm:h-full sm:px-6">
                <ArrowUpRight className="w-4 h-4" />
                Withdraw
              </Button>
            }
          />
        </div>

        <div className="bg-amber-500/10 border border-amber-500/30 rounded-xl p-4 text-sm text-amber-700 dark:text-amber-300">
          Deposits and withdrawals aren&apos;t connected to a real payment processor yet — no real money moves. Balances
          themselves are real and shared across your sessions.
        </div>

<div className="grid sm:grid-cols-3 gap-4">
          <div className="bg-card rounded-2xl border border-border p-5">
            <p className="text-sm text-muted-foreground">Total earned</p>
            <p className="text-2xl font-bold text-foreground">{(wallet?.total_earned || 0).toFixed(2)}</p>
          </div>
          <div className="bg-card rounded-2xl border border-border p-5">
            <p className="text-sm text-muted-foreground">Total spent</p>
            <p className="text-2xl font-bold text-foreground">{(wallet?.total_spent || 0).toFixed(2)}</p>
          </div>
          <div className="bg-card rounded-2xl border border-border p-5">
            <p className="text-sm text-muted-foreground">In escrow</p>
            <p className="text-2xl font-bold text-foreground">{(wallet?.escrow_balance || 0).toFixed(2)}</p>
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
