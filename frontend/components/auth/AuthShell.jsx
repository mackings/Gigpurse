import Link from "next/link";
import { Disc3 } from "lucide-react";

export default function AuthShell({ children }) {
  return (
    <div className="min-h-screen bg-background relative flex items-center justify-center p-4 overflow-hidden">
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_70%_50%_at_50%_0%,var(--accent),transparent)] opacity-60" />
      <div className="relative max-w-md w-full">
        <Link href="/" className="flex items-center justify-center gap-2.5 mb-8">
          <div className="w-9 h-9 bg-primary rounded-xl flex items-center justify-center">
            <Disc3 className="w-5 h-5 text-primary-foreground" strokeWidth={2.25} />
          </div>
          <span className="text-xl font-bold text-foreground tracking-tight">GigPurse</span>
        </Link>
        {children}
      </div>
    </div>
  );
}
