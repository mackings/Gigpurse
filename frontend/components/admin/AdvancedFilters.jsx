"use client";

import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Button } from "@/components/ui/button";
import { SlidersHorizontal, X } from "lucide-react";

// Shared "Advanced filters" popover for admin list pages — a button with an
// active-count badge that opens a small form; each page supplies its own
// fields as children and an activeCount/onClear for the reset action.
export default function AdvancedFilters({
  activeCount = 0,
  onClear,
  children,
}) {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" className="gap-2 relative">
          <SlidersHorizontal className="w-4 h-4" />
          Advanced
          {activeCount > 0 && (
            <span className="ml-0.5 inline-flex items-center justify-center w-4.5 h-4.5 rounded-full bg-primary text-primary-foreground text-[10px] font-semibold">
              {activeCount}
            </span>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-80 p-4 space-y-3">
        <div className="flex items-center justify-between">
          <p className="text-sm font-semibold text-foreground">
            Advanced filters
          </p>
          {activeCount > 0 && (
            <button
              onClick={onClear}
              className="text-xs text-muted-foreground hover:text-foreground flex items-center gap-1"
            >
              <X className="w-3 h-3" />
              Clear
            </button>
          )}
        </div>
        {children}
      </PopoverContent>
    </Popover>
  );
}
