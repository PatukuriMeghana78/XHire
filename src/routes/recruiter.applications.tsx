import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Users } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { EmptyState } from "@/components/common/EmptyState";
import { MatchScore } from "@/components/common/MatchScore";
import { PageHeader } from "@/components/common/PageHeader";
import { QueryBoundary } from "@/components/common/QueryBoundary";
import { applicationStatusLabel } from "@/components/common/StatusBadge";
import { Card, CardContent } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { formatDate } from "@/lib/format";
import { applicationService } from "@/services";
import type { ApplicationStatus } from "@/types";

const STATUSES: ApplicationStatus[] = [
  "applied",
  "under-review",
  "shortlisted",
  "interview",
  "rejected",
  "hired",
];

export const Route = createFileRoute("/recruiter/applications")({
  head: () => ({
    meta: [
      { title: "Applications — XHire" },
      { name: "description", content: "Every applicant across your roles, with match scores." },
      { property: "og:title", content: "Applications — XHire" },
      { property: "og:description", content: "Review and progress applicants on XHire." },
    ],
  }),
  component: RecruiterApplicationsPage,
});

function RecruiterApplicationsPage() {
  const queryClient = useQueryClient();
  const [filter, setFilter] = useState<ApplicationStatus | "all">("all");

  const applications = useQuery({
    queryKey: ["recruiter", "applications"],
    queryFn: () => applicationService.listRecruiterApplications(),
  });

  const setStatus = useMutation({
    mutationFn: ({ id, status }: { id: string; status: ApplicationStatus }) =>
      applicationService.updateStatus(id, status),
    onSuccess: () => {
      toast.success("Status updated");
      queryClient.invalidateQueries({ queryKey: ["recruiter", "applications"] });
    },
    onError: (error) => toast.error((error as Error).message || "Could not update status"),
  });

  const list = (applications.data ?? []).filter((a) => filter === "all" || a.status === filter);

  return (
    <>
      <PageHeader
        title="Applications"
        description="All applicants across your job postings."
        actions={
          <Select value={filter} onValueChange={(v) => setFilter(v as ApplicationStatus | "all")}>
            <SelectTrigger className="w-44" aria-label="Filter by status">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All statuses</SelectItem>
              {STATUSES.map((status) => (
                <SelectItem key={status} value={status}>
                  {applicationStatusLabel(status)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        }
      />

      <QueryBoundary isLoading={applications.isLoading} error={applications.error} onRetry={applications.refetch}>
        {list.length ? (
          <div className="space-y-4">
            {list.map((application) => (
              <Card key={application.id}>
                <CardContent className="grid gap-4 p-5 sm:grid-cols-[auto_minmax(0,1fr)_auto] sm:items-center">
                  <MatchScore score={application.matchScore ?? 0} size={56} />
                  <div className="min-w-0">
                    <Link
                      to="/recruiter/candidates/$candidateId"
                      params={{ candidateId: application.candidateId }}
                      className="truncate font-medium hover:underline"
                    >
                      {application.candidateName}
                    </Link>
                    <p className="truncate text-sm text-muted-foreground">
                      {application.jobTitle} · applied {formatDate(application.appliedAt)}
                    </p>
                  </div>
                  <Select
                    value={application.status}
                    onValueChange={(value) =>
                      setStatus.mutate({ id: application.id, status: value as ApplicationStatus })
                    }
                  >
                    <SelectTrigger className="w-44" aria-label="Change application status">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {STATUSES.map((status) => (
                        <SelectItem key={status} value={status}>
                          {applicationStatusLabel(status)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </CardContent>
              </Card>
            ))}
          </div>
        ) : (
          <EmptyState icon={Users} title="No applications match this filter" />
        )}
      </QueryBoundary>
    </>
  );
}
