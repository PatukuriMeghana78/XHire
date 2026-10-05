import { Link, useNavigate } from "@tanstack/react-router";
import {
  Briefcase,
  FileText,
  LayoutDashboard,
  LogOut,
  Menu,
  Sparkles,
  UserRound,
  Users,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { useState } from "react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { initials } from "@/lib/format";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/utils";
import type { UserRole } from "@/types";

interface NavItem {
  to: string;
  label: string;
  icon: LucideIcon;
  exact?: boolean;
}

const CANDIDATE_NAV: NavItem[] = [
  { to: "/candidate", label: "Dashboard", icon: LayoutDashboard, exact: true },
  { to: "/candidate/jobs", label: "Find jobs", icon: Briefcase },
  { to: "/candidate/applications", label: "Applications", icon: FileText },
  { to: "/candidate/resume", label: "Resume & AI", icon: Sparkles },
  { to: "/candidate/profile", label: "Profile", icon: UserRound },
];

const RECRUITER_NAV: NavItem[] = [
  { to: "/recruiter", label: "Dashboard", icon: LayoutDashboard, exact: true },
  { to: "/recruiter/jobs", label: "Job postings", icon: Briefcase },
  { to: "/recruiter/applications", label: "Applications", icon: Users },
  { to: "/recruiter/profile", label: "Company profile", icon: UserRound },
];

function NavLinks({ role, onNavigate }: { role: UserRole; onNavigate?: () => void }) {
  const items = role === "recruiter" ? RECRUITER_NAV : CANDIDATE_NAV;
  return (
    <nav className="space-y-1">
      {items.map(({ to, label, icon: Icon, exact }) => (
        <Link
          key={to}
          to={to}
          activeOptions={{ exact: exact ?? false }}
          onClick={onNavigate}
          className="flex items-center gap-3 rounded-lg px-3 py-2 text-sm text-sidebar-foreground/75 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground data-[status=active]:bg-sidebar-accent data-[status=active]:font-medium data-[status=active]:text-sidebar-primary"
        >
          <Icon className="h-4 w-4 shrink-0" aria-hidden />
          <span className="truncate">{label}</span>
        </Link>
      ))}
    </nav>
  );
}

function Brand() {
  return (
    <Link to="/" className="flex items-center gap-2 px-3 py-1">
      <span className="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-sidebar-primary font-display text-sm font-bold text-sidebar-primary-foreground">
        X
      </span>
      <span className="font-display text-lg font-semibold text-sidebar-foreground">XHire</span>
    </Link>
  );
}

export function DashboardLayout({ role, children }: { role: UserRole; children: ReactNode }) {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);

  async function handleLogout() {
    await logout();
    navigate({ to: "/login", replace: true });
  }

  return (
    <div className="min-h-screen bg-background lg:grid lg:grid-cols-[256px_minmax(0,1fr)]">
      <aside className="sticky top-0 hidden h-screen flex-col justify-between bg-sidebar p-4 lg:flex">
        <div className="space-y-6">
          <Brand />
          <NavLinks role={role} />
        </div>
        <div className="space-y-3 border-t border-sidebar-border pt-4">
          <div className="flex min-w-0 items-center gap-3 px-1">
            <span className="grid h-9 w-9 shrink-0 place-items-center rounded-full bg-sidebar-accent text-xs font-semibold text-sidebar-accent-foreground">
              {initials(user?.fullName)}
            </span>
            <div className="min-w-0">
              <p className="truncate text-sm text-sidebar-foreground">{user?.fullName}</p>
              <p className="truncate text-xs capitalize text-sidebar-foreground/60">{role}</p>
            </div>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleLogout}
            className="w-full justify-start text-sidebar-foreground/75 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
          >
            <LogOut className="mr-2 h-4 w-4" aria-hidden /> Sign out
          </Button>
        </div>
      </aside>

      <div className="flex min-w-0 flex-col">
        <header className="sticky top-0 z-30 flex items-center justify-between gap-3 border-b bg-background/85 px-4 py-3 backdrop-blur lg:hidden">
          <Sheet open={open} onOpenChange={setOpen}>
            <SheetTrigger asChild>
              <Button variant="ghost" size="icon" aria-label="Open navigation">
                <Menu className="h-5 w-5" />
              </Button>
            </SheetTrigger>
            <SheetContent side="left" className="w-64 bg-sidebar p-4">
              <SheetTitle className="sr-only">Navigation</SheetTitle>
              <div className="space-y-6">
                <Brand />
                <NavLinks role={role} onNavigate={() => setOpen(false)} />
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleLogout}
                  className="w-full justify-start text-sidebar-foreground/75"
                >
                  <LogOut className="mr-2 h-4 w-4" aria-hidden /> Sign out
                </Button>
              </div>
            </SheetContent>
          </Sheet>
          <span className="font-display text-base font-semibold">XHire</span>
          <span className={cn("grid h-8 w-8 place-items-center rounded-full bg-muted text-xs font-semibold")}>
            {initials(user?.fullName)}
          </span>
        </header>

        <main className="mx-auto w-full max-w-6xl flex-1 space-y-8 px-4 py-6 sm:px-6 lg:py-10">
          {children}
        </main>
      </div>
    </div>
  );
}
