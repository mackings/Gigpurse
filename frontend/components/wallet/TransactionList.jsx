import { ArrowDownLeft, ArrowUpRight, Lock, Unlock, Receipt } from "lucide-react";
import { formatMoney } from "@/lib/utils";

const typeMeta = {
  deposit: { label: "Deposit", icon: ArrowDownLeft, sign: "+", color: "bg-emerald-500" },
  withdrawal: { label: "Withdrawal", icon: ArrowUpRight, sign: "-", color: "bg-rose-500" },
  escrow_hold: { label: "Escrow funded", icon: Lock, sign: "-", color: "bg-violet-500" },
  escrow_release: { label: "Escrow released", icon: Unlock, sign: "-", color: "bg-violet-500" },
  payment_received: { label: "Payment received", icon: ArrowDownLeft, sign: "+", color: "bg-emerald-500" },
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
        const meta = typeMeta[tx.type] || { label: tx.type, icon: Receipt, sign: "", color: "bg-muted-foreground" };
        const Icon = meta.icon;
        return (
          <div
            key={tx.id}
            className="group flex items-center justify-between gap-3 p-3 rounded-xl bg-muted/40 transition-all duration-200 hover:bg-muted hover:shadow-sm"
          >
            <div className="flex items-center gap-3 min-w-0">
              <div className={`w-9 h-9 rounded-lg flex items-center justify-center shrink-0 shadow-sm ${meta.color}`}>
                <Icon className="w-4 h-4 text-white" />
              </div>
              <div className="min-w-0">
                <p className="text-sm font-medium text-foreground truncate">{meta.label}</p>
                <p className="text-xs text-muted-foreground truncate">{tx.description}</p>
              </div>
            </div>
            <div className="text-right shrink-0">
              <p className={`text-sm font-semibold tabular-nums ${meta.sign === "+" ? "text-emerald-600 dark:text-emerald-400" : "text-foreground"}`}>
                {meta.sign}
                {formatMoney(tx.amount)}
              </p>
              <p className="text-xs text-muted-foreground">{new Date(tx.created_at).toLocaleDateString()}</p>
            </div>
          </div>
        );
      })}
    </div>
  );
}
