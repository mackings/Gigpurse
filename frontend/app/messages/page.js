import { Suspense } from "react";
import MessagesView from "./messages-view";

export default function MessagesPage() {
  return (
    <Suspense>
      <MessagesView />
    </Suspense>
  );
}
