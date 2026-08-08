"use client";

import { useCurrentUser } from "@/hooks/use-current-user";
import { useWallet, useEscrowHoldings } from "@/hooks/use-wallet";
import WalletCard from "@/components/wallet/WalletCard";
import TransactionList from "@/components/wallet/TransactionList";
import EscrowHoldingsList from "@/components/wallet/EscrowHoldingsList";
import { formatMoney } from "@/lib/utils";
import { Loader2, ShieldCheck } from "lucide-react";

// Tailwind's JIT scans for literal class strings, so each accent needs its
// classes spelled out in full here rather than interpolated (`bg-${accent}-500`
// would never get generated into the build's CSS).
const STAT_ACCENTS = {
  emerald: { border: "hover:border-emerald-500/20", glow: "bg-emerald-500" },
  rose: { border: "hover:border-rose-500/20", glow: "bg-rose-500" },
};

function StatCard({ label, value, accent }) {
  const { border, glow } = STAT_ACCENTS[accent];
  return (
    <div
      className={`group relative overflow-hidden bg-card rounded-2xl border border-border p-5 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:-translate-y-0.5 ${border}`}
    >
      <div className={`absolute -top-6 -right-6 w-24 h-24 rounded-full opacity-[0.08] blur-2xl transition-opacity duration-300 group-hover:opacity-[0.16] ${glow}`} />
      <p className="relative text-sm text-muted-foreground">{label}</p>
      <p className="relative text-2xl font-bold text-foreground tabular-nums">{formatMoney(value || 0, { decimals: 2 })}</p>
    </div>
  );
}

// A client never receives a payout, so the musician's payout-account card
// and "total earned" stat are meaningless here — instead this is "how much
// of your money is currently tied up in escrow, and can I get it back."
function ClientWallet() {
  const { wallet, transactions } = useWallet();
  const { holdings, requestRefund } = useEscrowHoldings();
  const totalInEscrow = holdings.reduce((sum, h) => sum + h.amount, 0);

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground tracking-tight">Wallet</h1>
        <p className="text-muted-foreground">Money you currently have in escrow, and your payment history.</p>
      </div>

      <div className="group relative overflow-hidden bg-primary rounded-2xl p-6 text-primary-foreground shadow-sm">
        <div className="flex items-center gap-2 mb-2 opacity-80">
          <ShieldCheck className="w-5 h-5" />
          <span className="text-sm font-medium">Currently in escrow</span>
        </div>
        <div className="text-3xl font-bold tabular-nums">{formatMoney(totalInEscrow, { decimals: 2 })}</div>
        <p className="text-sm opacity-80 mt-1">
          {holdings.length > 0
            ? `Held across ${holdings.length} payment${holdings.length === 1 ? "" : "s"} until work is delivered or you request a refund.`
            : "Nothing held right now — money moves into escrow the moment you hire someone or fund a milestone."}
        </p>
      </div>

      <StatCard label="Total spent" value={wallet?.total_spent} accent="rose" />

      <div className="bg-card rounded-2xl border border-border p-6">
        <h2 className="font-semibold text-foreground mb-4">In escrow</h2>
        <EscrowHoldingsList holdings={holdings} onRequestRefund={requestRefund} />
      </div>

      <div className="bg-card rounded-2xl border border-border p-6">
        <h2 className="font-semibold text-foreground mb-4">Transaction history</h2>
        <TransactionList transactions={transactions} />
      </div>
    </div>
  );
}

function MusicianWallet() {
  const { wallet, transactions } = useWallet();

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground tracking-tight">Wallet</h1>
        <p className="text-muted-foreground">Your payout account and payment history.</p>
      </div>

      <WalletCard payoutAccount={wallet?.payout_account} />

      <div className="grid sm:grid-cols-2 gap-4">
        <StatCard label="Total earned" value={wallet?.total_earned} accent="emerald" />
        <StatCard label="Total spent" value={wallet?.total_spent} accent="rose" />
      </div>

      <div className="bg-card rounded-2xl border border-border p-6">
        <h2 className="font-semibold text-foreground mb-4">Transaction history</h2>
        <TransactionList transactions={transactions} />
      </div>
    </div>
  );
}

export default function WalletPage() {
  const { user, isLoading } = useCurrentUser();

  if (isLoading || !user) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background py-12 px-4">
      {user.role === "client" ? <ClientWallet /> : <MusicianWallet />}
    </div>
  );
}
