import { ArrowDownLeft, ArrowUpRight, Lock, Unlock, Receipt } from "lucide-react";

const typeMeta = {
  deposit: { label: "Deposit", icon: ArrowDownLeft, sign: "+" },
  withdrawal: { label: "Withdrawal", icon: ArrowUpRight, sign: "-" },
  escrow_hold: { label: "Escrow funded", icon: Lock, sign: "-" },
  escrow_release: { label: "Escrow released", icon: Unlock, sign: "-" },
  payment_received: { label: "Payment received", icon: ArrowDownLeft, sign: "+" },
};

export default function TransactionList({ transactions }) {
  if (!transactions.length) {
    return (
      <div className="text-center py-12 text-muted-foreground">
        <Receipt className="w-8 h-8 mx-auto mb-2" />
        <p className="text-sm">No transactions yet.</p>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {transactions.map((tx) => {
        const meta = typeMeta[tx.type] || { label: tx.type, icon: Receipt, sign: "" };
        const Icon = meta.icon;
        return (
          <div key={tx.id} className="flex items-center justify-between gap-3 p-3 rounded-lg bg-muted/40">
            <div className="flex items-center gap-3 min-w-0">
              <div className="w-9 h-9 rounded-lg bg-muted flex items-center justify-center shrink-0">
                <Icon className="w-4 h-4 text-muted-foreground" />
              </div>
              <div className="min-w-0">
                <p className="text-sm font-medium text-foreground truncate">{meta.label}</p>
                <p className="text-xs text-muted-foreground truncate">{tx.description}</p>
              </div>
            </div>
            <div className="text-right shrink-0">
              <p className="text-sm font-semibold text-foreground">
                {meta.sign}
                {tx.amount}
              </p>
              <p className="text-xs text-muted-foreground">{new Date(tx.created_at).toLocaleDateString()}</p>
            </div>
          </div>
        );
      })}
    </div>
  );
}
