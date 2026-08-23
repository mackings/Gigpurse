"use client";

import { useParams } from "next/navigation";
import PendingPaymentView from "../pending-view";

// PayPetal appends "?status=...&txnId=..." to whatever redirect URL we give
// it without checking for an existing "?" — so the reference travels in the
// path here, not a query param (see milestone_usecase.go's Fund), keeping
// PayPetal's own query string the only one on the URL.
export default function PendingPaymentPathPage() {
  const params = useParams();
  const reference = params?.reference ? decodeURIComponent(params.reference) : null;
  return <PendingPaymentView reference={reference} />;
}
