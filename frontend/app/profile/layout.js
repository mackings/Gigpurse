"use client";

import AccountSettingsHeader from "@/components/account/AccountSettingsHeader";

export default function ProfileLayout({ children }) {
  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-4xl mx-auto px-4 py-12">
        <AccountSettingsHeader />
        {children}
      </div>
    </div>
  );
}
