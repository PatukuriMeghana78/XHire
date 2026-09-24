import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { FileText } from "lucide-react";
import { useState } from "react";

import { EmptyState } from "@/components/common/EmptyState";
import { MatchScore } from "@/components/common/MatchScore";
import { PageHeader } from "@/components/common/PageHeader";
import { QueryBoundary } from "@/components/common/QueryBoundary";
import { StatusBadge, applicationStatusLabel } from "@/components/common/StatusBadge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { formatDate } from "@/lib/format";
import { applicationService } from "@/services";
import type { ApplicationStatus } from "@/types";

const FILTERS: (ApplicationStatus | "all")[] = [
  "all",
  "applied",
  "under-review",
  "shortlisted",
  "interview",
  "rejected",
];

export const Route = createFileRoute("/candidate/applications")({
  head: () => ({
    meta: [
      { title: "My applications — XHire" },
      { name: "description", content: "Track every application and its current stage." },
      { property: "og:title", content: "My applications — XHire" },
      { property: "og:description", content: "Track application status on XHire." },
    ],
  }),
  component: ApplicationsPage,
});

function ApplicationsPage() {
  const [filter, setFilter] = useState<ApplicationStatus | "all">("all");
  const applications = useQuery({
    queryKey: ["applications", "mine"],
    queryFn: () => applicationService.listMyApplications(),
  });

  const list = (applications.data ?? []).filter((a) => filter === "all" || a.status === filter);

  return (
    <>
      <PageHeader title="My applications" description="Every role you've applied to, with its latest stage." />

      <Tabs value={filter} onValueChange={(v) => setFilter(v as ApplicationStatus | "all")}>
        <TabsList className="flex w-full flex-wrap justify-start">
          {FILTERS.map((value) => (
            <TabsTrigger key={value} value={value} className="capitalize">
              {value === "all" ? "All" : applicationStatusLabel(value)}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>

      <QueryBoundary isLoading={applications.isLoading} error={applications.error} onRetry={applications.refetch}>
        {list.length ? (
          <div className="space-y-4">
            {list.map((application) => (
              <Card key={application.id}>
                <CardContent className="grid gap-4 p-5 sm:grid-cols-[auto_minmax(0,1fr)_auto] sm:items-center">
                  <MatchScore score={application.matchScore ?? 0} size={64} />
                  <div className="min-w-0 space-y-2">
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="truncate font-medium">{application.jobTitle}</p>
                      <StatusBadge status={application.status} />
                    </div>
                    <p className="text-sm text-muted-foreground">
                      {application.companyName} · applied {formatDate(application.appliedAt)}
                    </p>
                    {application.timeline?.length ? (
                      <ol className="space-y-1 border-l pl-4 text-xs text-muted-foreground">
                        {application.timeline.map((step) => (
                          <li key={`${step.status}-${step.at}`} className="relative">
                            <span className="absolute -left-[21px] top-1.5 h-1.5 w-1.5 rounded-full bg-border" />
                            <span className="font-medium text-foreground">
                              {applicationStatusLabel(step.status)}
                            </span>{" "}
                            · {formatDate(step.at)}
                            {step.note ? ` — ${step.note}` : ""}
                          </li>
                        ))}
                      </ol>
                    ) : null}
                  </div>
                  <Button asChild variant="outline" size="sm" className="justify-self-start sm:justify-self-end">
                    <Link to="/candidate/jobs/$jobId" params={{ jobId: application.jobId }}>
                      View role
                    </Link>
                  </Button>
                </CardContent>
              </Card>
            ))}
          </div>
        ) : (
          <EmptyState
            icon={FileText}
            title="Nothing here yet"
            description="Applications you submit will appear in this list."
            action={
              <Button asChild size="sm">
                <Link to="/candidate/jobs">Browse jobs</Link>
              </Button>
            }
          />
        )}
      </QueryBoundary>
    </>
  );
}
