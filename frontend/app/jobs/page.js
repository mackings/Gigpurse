import { Suspense } from "react";
import JobsView from "./jobs-view";

export default function JobsPage() {
  return (
    <Suspense>
      <JobsView />
    </Suspense>
  );
}
