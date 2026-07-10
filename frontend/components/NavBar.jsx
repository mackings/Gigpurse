"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, usePathname } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useRealtime } from "@/lib/RealtimeProvider";
import { logout as apiLogout } from "@/lib/api";
import { Button } from "@/components/ui/button";
import ThemeToggle from "@/components/ThemeToggle";
import NotificationBell from "@/components/notifications/NotificationBell";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Disc3,
  Menu,
  X,
  User,
  UserCog,
  LogOut,
  LayoutDashboard,
  Search,
  MessageCircle,
  Wallet,
  Briefcase,
  FolderOpen,
  ClipboardList,
  CalendarCheck,
  ShieldAlert,
  ShieldCheck,
} from "lucide-react";

export default function NavBar() {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const { user, isLoading, isAuthenticated, refetch } = useCurrentUser();
  const { unreadMessageCount } = useRealtime();
  const router = useRouter();
  const pathname = usePathname();
  const queryClient = useQueryClient();

  // NavBar lives in the shared layout and never remounts between routes, so
  // isMenuOpen otherwise survives a client-side navigation — e.g. open the
  // mobile menu, tap "Sign in", log in, land on /jobs with the drawer still
  // covering the page. Closing on every route change is a hard guarantee,
  // not dependent on every link inside the drawer remembering to do it.
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setIsMenuOpen(false);
  }, [pathname]);

  const isTalent = user?.role === "musician";
  // Talent land on the job board, not the stats dashboard — it's their
  // entry point for browsing, saving, and applying to gigs.
  const dashboardUrl = isTalent ? "/jobs" : "/dashboard/client";
  // Talent edit their extended profile through the onboarding wizard;
  // clients get the simpler single-page form at /profile itself.
  const profileHref = isTalent ? "/onboarding" : "/profile";
  // Moderators only have access to /admin/disputes; admins get the full dashboard.
  const canSeeAdminLink = user?.role === "admin" || user?.role === "moderator";
  const adminHref = user?.role === "moderator" ? "/admin/disputes" : "/admin";
  const adminLabel = user?.role === "moderator" ? "Disputes queue" : "Admin dashboard";

  async function handleLogout() {
    await apiLogout();
    queryClient.clear();
    refetch();
    router.push("/");
  }

  return (
    <nav className="sticky top-0 z-50 bg-background/90 backdrop-blur-md border-b border-border">
      <div className="max-w-7xl mx-auto px-4 sm:px-6">
        <div className="flex items-center justify-between h-16">
          <Link href="/" className="flex items-center gap-2.5">
            <div className="w-8 h-8 bg-primary rounded-lg flex items-center justify-center">
              <Disc3 className="w-4 h-4 text-primary-foreground" strokeWidth={2.25} />
            </div>
            <span className="text-lg font-bold text-foreground tracking-tight">GigPurse</span>
          </Link>

          <div className="hidden md:flex items-center gap-1">
            <Link
              href="/browse"
              className="px-3 py-2 rounded-lg text-sm text-muted-foreground hover:text-foreground hover:bg-accent font-medium transition-colors"
            >
              Browse Talent
            </Link>

            {!isLoading && (
              <>
                {isAuthenticated ? (
                  <div className="flex items-center gap-1 ml-2">
                    <Link href={dashboardUrl}>
                      <Button variant="ghost" className="gap-2 text-muted-foreground hover:text-foreground">
                        {isTalent ? <Briefcase className="w-4 h-4" /> : <LayoutDashboard className="w-4 h-4" />}
                        {isTalent ? "Find Gigs" : "Dashboard"}
                      </Button>
                    </Link>

                    <Link href="/messages">
                      <Button variant="ghost" size="icon" className="relative text-muted-foreground hover:text-foreground">
                        <MessageCircle className="w-4 h-4" />
                        {unreadMessageCount > 0 && (
                          <span className="absolute -top-0.5 -right-0.5 w-4 h-4 rounded-full bg-primary text-primary-foreground text-[10px] font-semibold flex items-center justify-center">
                            {unreadMessageCount > 9 ? "9+" : unreadMessageCount}
                          </span>
                        )}
                      </Button>
                    </Link>

                    <Link href="/wallet">
                      <Button variant="ghost" size="icon" className="text-muted-foreground hover:text-foreground">
                        <Wallet className="w-4 h-4" />
                      </Button>
                    </Link>

                    {isTalent && (
                      <Link href="/dashboard/talent">
                        <Button
                          variant="ghost"
                          size="icon"
                          title="My stats"
                          className="text-muted-foreground hover:text-foreground"
                        >
                          <LayoutDashboard className="w-4 h-4" />
                        </Button>
                      </Link>
                    )}

                    <NotificationBell className="text-muted-foreground hover:text-foreground" />
                    <ThemeToggle className="text-muted-foreground hover:text-foreground" />

                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="outline" size="icon" className="ml-1 rounded-full">
                          <User className="w-4 h-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" className="w-48">
                        <div className="px-2 py-1.5">
                          <p className="text-sm font-medium">{user?.name}</p>
                          <p className="text-xs text-muted-foreground">{user?.email}</p>
                          <p className="text-xs text-primary capitalize mt-0.5 font-medium">{isTalent ? "Talent" : user?.role}</p>
                        </div>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem asChild>
                          <Link href={profileHref}>
                            <UserCog className="w-4 h-4 mr-2" />
                            Profile
                          </Link>
                        </DropdownMenuItem>
                        {isTalent && (
                          <DropdownMenuItem asChild>
                            <Link href="/profile/portfolio">
                              <FolderOpen className="w-4 h-4 mr-2" />
                              Portfolio
                            </Link>
                          </DropdownMenuItem>
                        )}
                        {isTalent && (
                          <DropdownMenuItem asChild>
                            <Link href="/profile/jobs">
                              <ClipboardList className="w-4 h-4 mr-2" />
                              My Jobs
                            </Link>
                          </DropdownMenuItem>
                        )}
                        <DropdownMenuItem asChild>
                          <Link href="/profile/bookings">
                            <CalendarCheck className="w-4 h-4 mr-2" />
                            Bookings
                          </Link>
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem asChild>
                          <Link href="/disputes">
                            <ShieldAlert className="w-4 h-4 mr-2" />
                            Disputes
                          </Link>
                        </DropdownMenuItem>
                        {canSeeAdminLink && (
                          <DropdownMenuItem asChild>
                            <Link href={adminHref}>
                              <ShieldCheck className="w-4 h-4 mr-2" />
                              {adminLabel}
                            </Link>
                          </DropdownMenuItem>
                        )}
                        <DropdownMenuSeparator />
                        <DropdownMenuItem onClick={handleLogout} className="text-destructive focus:text-destructive">
                          <LogOut className="w-4 h-4 mr-2" />
                          Logout
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                ) : (
                  <div className="flex items-center gap-2 ml-2">
                    <ThemeToggle className="text-muted-foreground hover:text-foreground" />
                    <Link href="/login">
                      <Button variant="ghost" className="text-muted-foreground hover:text-foreground">
                        Sign in
                      </Button>
                    </Link>
                    <Link href="/role-selection">
                      <Button>Create account</Button>
                    </Link>
                  </div>
                )}
              </>
            )}
          </div>

          <div className="flex items-center gap-1 md:hidden">
            {isAuthenticated && <NotificationBell className="text-muted-foreground" />}
            <ThemeToggle className="text-muted-foreground" />
            <button className="p-2 text-foreground" onClick={() => setIsMenuOpen(!isMenuOpen)}>
              {isMenuOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
            </button>
          </div>
        </div>
      </div>

      {isMenuOpen && (
        <div className="md:hidden border-t border-border bg-background">
          <div className="px-4 py-4 space-y-1">
            <Link
              href="/browse"
              className="flex items-center gap-2 p-2.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent"
              onClick={() => setIsMenuOpen(false)}
            >
              <Search className="w-4 h-4" />
              Browse Talent
            </Link>

            {isAuthenticated ? (
              <>
                <Link
                  href={dashboardUrl}
                  className="flex items-center gap-2 p-2.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent"
                  onClick={() => setIsMenuOpen(false)}
                >
                  {isTalent ? <Briefcase className="w-4 h-4" /> : <LayoutDashboard className="w-4 h-4" />}
                  {isTalent ? "Find Gigs" : "Dashboard"}
                </Link>
                <Link
                  href="/messages"
                  className="flex items-center gap-2 p-2.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent"
                  onClick={() => setIsMenuOpen(false)}
                >
                  <MessageCircle className="w-4 h-4" />
                  Messages
                  {unreadMessageCount > 0 && (
                    <span className="ml-auto w-5 h-5 rounded-full bg-primary text-primary-foreground text-[10px] font-semibold flex items-center justify-center">
                      {unreadMessageCount > 9 ? "9+" : unreadMessageCount}
                    </span>
                  )}
                </Link>
                <Link
                  href="/wallet"
                  className="flex items-center gap-2 p-2.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent"
                  onClick={() => setIsMenuOpen(false)}
                >
                  <Wallet className="w-4 h-4" />
                  Wallet
                </Link>
                {isTalent && (
                  <Link
                    href="/dashboard/talent"
                    className="flex items-center gap-2 p-2.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent"
                    onClick={() => setIsMenuOpen(false)}
                  >
                    <LayoutDashboard className="w-4 h-4" />
                    My stats
                  </Link>
                )}
                <Link
                  href={profileHref}
                  className="flex items-center gap-2 p-2.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent"
                  onClick={() => setIsMenuOpen(false)}
                >
                  <UserCog className="w-4 h-4" />
                  Profile
                </Link>
                {isTalent && (
                  <Link
                    href="/profile/portfolio"
                    className="flex items-center gap-2 p-2.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent"
                    onClick={() => setIsMenuOpen(false)}
                  >
                    <FolderOpen className="w-4 h-4" />
                    Portfolio
                  </Link>
                )}
                {isTalent && (
                  <Link
                    href="/profile/jobs"
                    className="flex items-center gap-2 p-2.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent"
                    onClick={() => setIsMenuOpen(false)}
                  >
                    <ClipboardList className="w-4 h-4" />
                    My Jobs
                  </Link>
                )}
                <Link
                  href="/profile/bookings"
                  className="flex items-center gap-2 p-2.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent"
                  onClick={() => setIsMenuOpen(false)}
                >
                  <CalendarCheck className="w-4 h-4" />
                  Bookings
                </Link>
                <Link
                  href="/disputes"
                  className="flex items-center gap-2 p-2.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent"
                  onClick={() => setIsMenuOpen(false)}
                >
                  <ShieldAlert className="w-4 h-4" />
                  Disputes
                </Link>
                {canSeeAdminLink && (
                  <Link
                    href={adminHref}
                    className="flex items-center gap-2 p-2.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent"
                    onClick={() => setIsMenuOpen(false)}
                  >
                    <ShieldCheck className="w-4 h-4" />
                    {adminLabel}
                  </Link>
                )}
                <button
                  onClick={handleLogout}
                  className="flex items-center gap-2 p-2.5 rounded-lg text-destructive hover:bg-accent w-full text-left"
                >
                  <LogOut className="w-4 h-4" />
                  Logout
                </button>
              </>
            ) : (
              <div className="space-y-2 pt-2">
                <Link href="/login" className="block" onClick={() => setIsMenuOpen(false)}>
                  <Button variant="outline" className="w-full">
                    Sign in
                  </Button>
                </Link>
                <Link href="/role-selection" className="block" onClick={() => setIsMenuOpen(false)}>
                  <Button className="w-full">Create account</Button>
                </Link>
              </div>
            )}
          </div>
        </div>
      )}
    </nav>
  );
}
