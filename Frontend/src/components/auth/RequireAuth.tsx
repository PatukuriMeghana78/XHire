import { Navigate } from "@tanstack/react-router";
import type { ReactNode } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { homeForRole, useAuth } from "@/lib/auth";
import type { UserRole } from "@/types";

export function RequireAuth({ role, children }: { role: UserRole; children: ReactNode }) {
  const { user, ready } = useAuth();

  if (!ready) {
    return (
      <div className="space-y-4 p-8">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!user) return <Navigate to="/login" search={{ role }} replace />;
  if (user.role !== role) return <Navigate to={homeForRole(user.role)} replace />;

  return <>{children}</>;
}
