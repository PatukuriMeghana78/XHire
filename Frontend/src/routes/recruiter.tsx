import { createFileRoute, Outlet } from "@tanstack/react-router";

import { RequireAuth } from "@/components/auth/RequireAuth";
import { DashboardLayout } from "@/components/layout/DashboardLayout";

export const Route = createFileRoute("/recruiter")({
  ssr: false,
  component: RecruiterLayout,
});

function RecruiterLayout() {
  return (
    <RequireAuth role="recruiter">
      <DashboardLayout role="recruiter">
        <Outlet />
      </DashboardLayout>
    </RequireAuth>
  );
}
