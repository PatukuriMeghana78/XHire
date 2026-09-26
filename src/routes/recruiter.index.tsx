import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Briefcase, Plus, Star, Users } from "lucide-react";

import { EmptyState } from "@/components/common/EmptyState";
import { MatchScore } from "@/components/common/MatchScore";
import { PageHeader } from "@/components/common/PageHeader";
import { QueryBoundary } from "@/components/common/QueryBoundary";
import { StatCard } from "@/components/common/StatCard";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatRelativeDate } from "@/lib/format";
import { useAuth } from "@/lib/auth";
import { applicationService, jobService } from "@/services";

export const Route = createFileRoute("/recruiter/")({
  head: () => ({
    meta: [
      { title: "Recruiter dashboard — XHire" },
      { name: "description", content: "Your open roles, applicant pipeline and ranked shortlists." },
      { property: "og:title", content: "Recruiter dashboard — XHire" },
      { property: "og:description", content: "Manage roles and ranked applicants." },
    ],
  }),
  component: RecruiterDashboard,
});

function RecruiterDashboard() {
  const { user } = useAuth();
  const jobs = useQuery({ queryKey: ["recruiter", "jobs"], queryFn: () => jobService.listRecruiterJobs() });
  const applications = useQuery({
    queryKey: ["recruiter", "applications"],
    queryFn: () => applicationService.listRecruiterApplications(),
  });

  const jobList = jobs.data ?? [];
  const appList = applications.data ?? [];
  const openRoles = jobList.filter((job) => job.status === "open").length;
  const shortlisted = appList.filter((a) => ["shortlisted", "interview"].includes(a.status)).length;

  return (
    <>
      <PageHeader
        title={`Hi ${user?.fullName?.split(" ")[0] ?? "there"}`}
        description="Your hiring pipeline at a glance."
        actions={
          <Button asChild>
            <Link to="/recruiter/jobs/new">
              <Plus className="mr-2 h-4 w-4" /> Post a job
            </Link>
          </Button>
        }
      />

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Job postings" value={jobList.length} icon={Briefcase} />
        <StatCard label="Open roles" value={openRoles} icon={Briefcase} />
        <StatCard label="Applicants" value={appList.length} icon={Users} />
        <StatCard label="Shortlisted" value={shortlisted} icon={Star} />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader className="flex-row items-center justify-between gap-3 space-y-0">
            <CardTitle className="font-display text-base">Your roles</CardTitle>
            <Button asChild variant="ghost" size="sm">
              <Link to="/recruiter/jobs">Manage</Link>
            </Button>
          </CardHeader>
          <CardContent>
            <QueryBoundary isLoading={jobs.isLoading} error={jobs.error} onRetry={jobs.refetch}>
              {jobList.length ? (
                <ul className="divide-y">
                  {jobList.slice(0, 5).map((job) => (
                    <li key={job.id} className="flex items-center justify-between gap-3 py-3">
                      <div className="min-w-0">
                        <Link
                          to="/recruiter/jobs/$jobId"
                          params={{ jobId: job.id }}
                          className="truncate text-sm font-medium hover:underline"
                        >
                          {job.title}
                        </Link>
                        <p className="truncate text-xs text-muted-foreground">
                          {job.applicantCount ?? 0} applicants · posted {formatRelativeDate(job.postedAt)}
                        </p>
                      </div>
                      <span className="shrink-0 text-xs capitalize text-muted-foreground">{job.status}</span>
                    </li>
                  ))}
                </ul>
              ) : (
                <EmptyState
                  icon={Briefcase}
                  title="No roles posted"
                  action={
                    <Button asChild size="sm">
                      <Link to="/recruiter/jobs/new">Post your first job</Link>
                    </Button>
                  }
                />
              )}
            </QueryBoundary>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex-row items-center justify-between gap-3 space-y-0">
            <CardTitle className="font-display text-base">Latest applicants</CardTitle>
            <Button asChild variant="ghost" size="sm">
              <Link to="/recruiter/applications">View all</Link>
            </Button>
          </CardHeader>
          <CardContent>
            <QueryBoundary isLoading={applications.isLoading} error={applications.error} onRetry={applications.refetch}>
              {appList.length ? (
                <ul className="divide-y">
                  {appList.slice(0, 5).map((application) => (
                    <li key={application.id} className="flex items-center gap-3 py-3">
                      <MatchScore score={application.matchScore ?? 0} size={48} label="" />
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium">{application.candidateName}</p>
                        <p className="truncate text-xs text-muted-foreground">{application.jobTitle}</p>
                      </div>
                      <StatusBadge status={application.status} />
                    </li>
                  ))}
                </ul>
              ) : (
                <EmptyState icon={Users} title="No applicants yet" />
              )}
            </QueryBoundary>
          </CardContent>
        </Card>
      </div>
    </>
  );
}
