"use client";

import { Landmark, ShieldCheck, ShieldAlert } from "lucide-react";
import AddPayoutAccountModal from "@/components/wallet/AddPayoutAccountModal";
import { Button } from "@/components/ui/button";

// Replaces the old spendable-balance card — there's no reloadable balance
// with PayPetal, every payment is its own checkout and every payout goes
// straight to a bank account. This is "where does your money land," not
// "how much do you have."
export default function WalletCard({ payoutAccount }) {
  return (
    <div className="bg-primary rounded-2xl p-6 text-primary-foreground shadow-sm">
      <div className="flex items-center gap-2 mb-4 opacity-80">
        <Landmark className="w-5 h-5" />
        <span className="text-sm font-medium">Payout account</span>
      </div>

      {payoutAccount ? (
        <>
          <div className="flex items-center gap-1.5 mb-1">
            <ShieldCheck className="w-4 h-4 text-emerald-300" />
            <span className="text-sm font-medium text-emerald-300">Ready to receive payouts</span>
          </div>
          <div className="text-2xl font-bold mb-1">{payoutAccount.account_name}</div>
          <p className="text-sm opacity-80 mb-5">
            {payoutAccount.bank_name} &middot; ****{payoutAccount.account_number?.slice(-4)}
          </p>
          <AddPayoutAccountModal
            trigger={
              <Button variant="secondary" size="sm">
                Change account
              </Button>
            }
          />
        </>
      ) : (
        <>
          <div className="flex items-center gap-1.5 mb-1">
            <ShieldAlert className="w-4 h-4 text-amber-300" />
            <span className="text-sm font-medium text-amber-300">No payout account yet</span>
          </div>
          <p className="text-sm opacity-80 mb-5">Add a bank account so clients can pay you and escrow can be released to you.</p>
          <AddPayoutAccountModal trigger={<Button variant="secondary">Add payout account</Button>} />
        </>
      )}
    </div>
  );
}
