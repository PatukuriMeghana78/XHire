import { createFileRoute, Outlet } from "@tanstack/react-router";

import { RequireAuth } from "@/components/auth/RequireAuth";
import { DashboardLayout } from "@/components/layout/DashboardLayout";

export const Route = createFileRoute("/candidate")({
  ssr: false,
  component: CandidateLayout,
});

function CandidateLayout() {
  return (
    <RequireAuth role="candidate">
      <DashboardLayout role="candidate">
        <Outlet />
      </DashboardLayout>
    </RequireAuth>
  );
}
