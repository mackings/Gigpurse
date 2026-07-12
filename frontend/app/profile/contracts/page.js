"use client";

import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import ContractRow from "@/components/dashboard/ContractRow";
import { Loader2 } from "lucide-react";

export default function ProfileContracts() {
  const { data: contracts, isLoading } = useQuery({
    queryKey: ["contracts"],
    queryFn: () => apiGet("/contracts"),
  });

  const active = contracts?.filter((c) => c.status === "active") || [];
  const completed = contracts?.filter((c) => c.status === "completed") || [];

  if (isLoading) {
    return (
      <div className="flex justify-center py-24">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-lg font-semibold text-foreground tracking-tight mb-4">Active contracts</h2>
        {active.length ? (
          <div className="space-y-3">
            {active.map((contract) => (
              <ContractRow key={contract.id} contract={contract} />
            ))}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">No active contracts.</p>
        )}
      </div>

      <div>
        <h2 className="text-lg font-semibold text-foreground tracking-tight mb-4">Completed contracts</h2>
        {completed.length ? (
          <div className="space-y-3">
            {completed.map((contract) => (
              <ContractRow key={contract.id} contract={contract} />
            ))}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">No completed contracts yet.</p>
        )}
      </div>
    </div>
  );
}
