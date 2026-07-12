"use client";

import * as React from "react";
import { Dialog as DialogPrimitive } from "radix-ui";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { XIcon } from "lucide-react";

function Sheet({ ...props }) {
  return <DialogPrimitive.Root data-slot="sheet" {...props} />;
}

function SheetTrigger({ ...props }) {
  return <DialogPrimitive.Trigger data-slot="sheet-trigger" {...props} />;
}

function SheetPortal({ ...props }) {
  return <DialogPrimitive.Portal data-slot="sheet-portal" {...props} />;
}

function SheetClose({ ...props }) {
  return <DialogPrimitive.Close data-slot="sheet-close" {...props} />;
}

function SheetOverlay({ className, ...props }) {
  return (
    <DialogPrimitive.Overlay
      data-slot="sheet-overlay"
      className={cn(
        "fixed inset-0 z-50 bg-black/30 duration-200 data-open:animate-in data-open:fade-in-0 data-closed:animate-out data-closed:fade-out-0",
        className
      )}
      {...props}
    />
  );
}

// Responsive job-detail panel: a bottom sheet on mobile (matches the app's
// installed-PWA feel — same interaction as every other mobile action
// sheet), a right-side panel on desktop (matches Upwork's job detail
// panel). Driven entirely by Tailwind breakpoints, not JS viewport checks,
// so it's correct on first paint with no layout shift.
function SheetContent({ className, children, showCloseButton = true, ...props }) {
  return (
    <SheetPortal>
      <SheetOverlay />
      <DialogPrimitive.Content
        data-slot="sheet-content"
        className={cn(
          "fixed z-50 flex flex-col bg-popover text-popover-foreground shadow-xl outline-none",
          "inset-x-0 bottom-0 max-h-[88vh] rounded-t-2xl border-t border-border",
          "data-open:animate-in data-open:slide-in-from-bottom data-closed:animate-out data-closed:slide-out-to-bottom",
          "sm:inset-y-0 sm:right-0 sm:left-auto sm:bottom-auto sm:h-full sm:max-h-none sm:w-full sm:max-w-xl sm:rounded-t-none sm:rounded-l-2xl sm:border-t-0 sm:border-l",
          "sm:data-open:slide-in-from-right sm:data-closed:slide-out-to-right",
          "duration-300",
          className
        )}
        {...props}
      >
        <div className="mx-auto mt-2.5 h-1.5 w-10 shrink-0 rounded-full bg-border sm:hidden" />
        {children}
        {showCloseButton && (
          <DialogPrimitive.Close data-slot="sheet-close" asChild>
            <Button variant="ghost" className="absolute top-3 right-3 hidden sm:inline-flex" size="icon-sm">
              <XIcon />
              <span className="sr-only">Close</span>
            </Button>
          </DialogPrimitive.Close>
        )}
      </DialogPrimitive.Content>
    </SheetPortal>
  );
}

function SheetHeader({ className, ...props }) {
  return <div data-slot="sheet-header" className={cn("flex flex-col gap-1 border-b border-border p-4 sm:p-5", className)} {...props} />;
}

function SheetFooter({ className, ...props }) {
  return (
    <div
      data-slot="sheet-footer"
      className={cn("flex flex-col gap-2 border-t border-border bg-muted/30 p-4 sm:p-5", className)}
      {...props}
    />
  );
}

function SheetTitle({ className, ...props }) {
  return <DialogPrimitive.Title data-slot="sheet-title" className={cn("font-heading text-lg leading-tight font-semibold", className)} {...props} />;
}

function SheetDescription({ className, ...props }) {
  return <DialogPrimitive.Description data-slot="sheet-description" className={cn("text-sm text-muted-foreground", className)} {...props} />;
}

export { Sheet, SheetClose, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetOverlay, SheetPortal, SheetTitle, SheetTrigger };
