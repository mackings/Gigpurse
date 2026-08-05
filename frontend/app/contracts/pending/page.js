import { Suspense } from "react";
import PendingPaymentView from "./pending-view";

export default function PendingPaymentPage() {
  return (
    <Suspense>
      <PendingPaymentView />
    </Suspense>
  );
}
