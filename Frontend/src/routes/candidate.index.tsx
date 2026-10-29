import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Briefcase, FileText, Sparkles, Target } from "lucide-react";

import { ExplanationPanel } from "@/components/ai/ExplanationPanel";
import { EmptyState } from "@/components/common/EmptyState";
import { GmailCard } from "@/components/email/GmailCard";
import { PageHeader } from "@/components/common/PageHeader";
import { QueryBoundary } from "@/components/common/QueryBoundary";
import { StatCard } from "@/components/common/StatCard";
import { StatusBadge } from "@/components/common/StatusBadge";
import { JobCard } from "@/components/jobs/JobCard";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatRelativeDate } from "@/lib/format";
import { useAuth } from "@/lib/auth";
import { aiService, applicationService, jobService } from "@/services";

/** Query the Gmail redirect can leave behind on this route. */
interface DashboardSearch {
  gmail?: "connected" | "error";
  reason?: string;
}

export const Route = createFileRoute("/candidate/")({
  head: () => ({
    meta: [
      { title: "Candidate dashboard — XHire" },
      { name: "description", content: "Your applications, match scores and recommended jobs." },
      { property: "og:title", content: "Candidate dashboard — XHire" },
      { property: "og:description", content: "Track applications and AI match insight." },
    ],
  }),
  validateSearch: (search: Record<string, unknown>): DashboardSearch => {
    const gmailParam = search["gmail"];
    const reasonParam = search["reason"];
    const parsed: DashboardSearch = {};
    if (gmailParam === "connected" || gmailParam === "error") parsed.gmail = gmailParam;
    if (typeof reasonParam === "string") parsed.reason = reasonParam;
    return parsed;
  },
  component: CandidateDashboard,
});

function CandidateDashboard() {
  const { user } = useAuth();
  const { gmail, reason } = Route.useSearch();
  const applications = useQuery({
    queryKey: ["applications", "mine"],
    queryFn: () => applicationService.listMyApplications(),
  });
  const jobs = useQuery({ queryKey: ["jobs", {}], queryFn: () => jobService.listJobs() });

  const list = applications.data ?? [];
  const recommended = (jobs.data ?? []).slice(0, 3);
  // The match panel explains a real job. The previous hardcoded "j-1" was a mock
  // fixture id, which the numeric-id backend endpoint rejected with HTTP 400.
  const matchJobId = recommended[0]?.id;
  const topMatch = useQuery({
    queryKey: ["ai", "match", matchJobId],
    queryFn: () => aiService.getJobMatchExplanation(matchJobId as string),
    enabled: Boolean(matchJobId),
  });
  const active = list.filter((a) => !["rejected", "hired"].includes(a.status)).length;
  const shortlisted = list.filter((a) => ["shortlisted", "interview"].includes(a.status)).length;
  const avgScore = list.length
    ? Math.round(list.reduce((sum, a) => sum + (a.matchScore ?? 0), 0) / list.length)
    : 0;

  return (
    <>
      <PageHeader
        title={`Welcome back, ${user?.fullName?.split(" ")[0] ?? "there"}`}
        description="Here's where your search stands today."
        actions={
          <Button asChild>
            <Link to="/candidate/jobs">Find jobs</Link>
          </Button>
        }
      />

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Applications" value={list.length} icon={FileText} />
        <StatCard label="Active" value={active} hint="Still in the pipeline" icon={Briefcase} />
        <StatCard label="Shortlisted" value={shortlisted} icon={Target} />
        <StatCard label="Avg. match" value={`${avgScore}%`} icon={Sparkles} />
      </div>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,1fr)]">
        <Card>
          <CardHeader className="flex-row items-center justify-between gap-3 space-y-0">
            <CardTitle className="font-display text-base">Recent applications</CardTitle>
            <Button asChild variant="ghost" size="sm">
              <Link to="/candidate/applications">View all</Link>
            </Button>
          </CardHeader>
          <CardContent>
            <QueryBoundary isLoading={applications.isLoading} error={applications.error} onRetry={applications.refetch}>
              {list.length ? (
                <ul className="divide-y">
                  {list.slice(0, 5).map((application) => (
                    <li key={application.id} className="flex items-center justify-between gap-3 py-3">
                      <div className="min-w-0">
                        <p className="truncate text-sm font-medium">{application.jobTitle}</p>
                        <p className="truncate text-xs text-muted-foreground">
                          {application.companyName} · {formatRelativeDate(application.appliedAt)}
                        </p>
                      </div>
                      <StatusBadge status={application.status} />
                    </li>
                  ))}
                </ul>
              ) : (
                <EmptyState
                  icon={FileText}
                  title="No applications yet"
                  description="Apply to a role and track its progress here."
                  action={
                    <Button asChild size="sm">
                      <Link to="/candidate/jobs">Browse jobs</Link>
                    </Button>
                  }
                />
              )}
            </QueryBoundary>
          </CardContent>
        </Card>

        <div className="space-y-6">
          <GmailCard outcome={gmail ? { state: gmail, message: reason } : undefined} />
          <QueryBoundary isLoading={topMatch.isLoading} error={topMatch.error} onRetry={topMatch.refetch}>
            {topMatch.data ? <ExplanationPanel explanation={topMatch.data} /> : null}
          </QueryBoundary>
        </div>
      </div>

      <section className="space-y-4">
        <div className="flex items-center justify-between gap-3">
          <h2 className="font-display text-lg font-semibold">Recommended for you</h2>
          <Button asChild variant="ghost" size="sm">
            <Link to="/candidate/jobs">See all jobs</Link>
          </Button>
        </div>
        <QueryBoundary isLoading={jobs.isLoading} error={jobs.error} onRetry={jobs.refetch}>
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {recommended.map((job) => (
              <JobCard key={job.id} job={job} />
            ))}
          </div>
        </QueryBoundary>
      </section>
    </>
  );
}
