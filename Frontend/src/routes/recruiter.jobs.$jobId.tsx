import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, Users } from "lucide-react";
import { toast } from "sonner";

import { EmptyState } from "@/components/common/EmptyState";
import { MatchScore } from "@/components/common/MatchScore";
import { PageHeader } from "@/components/common/PageHeader";
import { QueryBoundary } from "@/components/common/QueryBoundary";
import { StatusBadge } from "@/components/common/StatusBadge";
import { JobForm } from "@/components/jobs/JobForm";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { aiService, applicationService, jobService } from "@/services";
import type { ApplicationStatus, Job } from "@/types";

const STATUSES: ApplicationStatus[] = [
  "applied",
  "under-review",
  "shortlisted",
  "interview",
  "rejected",
  "hired",
];

export const Route = createFileRoute("/recruiter/jobs/$jobId")({
  head: () => ({
    meta: [
      { title: "Manage role — XHire" },
      { name: "description", content: "Edit the role, review applicants and AI candidate rankings." },
      { property: "og:title", content: "Manage role — XHire" },
      { property: "og:description", content: "Applicants and ranked shortlist for this role." },
    ],
  }),
  component: ManageJobPage,
});

function ManageJobPage() {
  const { jobId } = Route.useParams();
  const queryClient = useQueryClient();

  const job = useQuery({ queryKey: ["jobs", jobId], queryFn: () => jobService.getJob(jobId) });
  const applications = useQuery({
    queryKey: ["recruiter", "applications", jobId],
    queryFn: () => applicationService.listApplicationsForJob(jobId),
  });
  const ranking = useQuery({
    queryKey: ["ai", "ranking", jobId],
    queryFn: () => aiService.getCandidateRanking(jobId),
  });

  const update = useMutation({
    mutationFn: (payload: Partial<Job>) => jobService.updateJob(jobId, payload),
    onSuccess: () => {
      toast.success("Job updated");
      queryClient.invalidateQueries({ queryKey: ["jobs", jobId] });
      queryClient.invalidateQueries({ queryKey: ["recruiter", "jobs"] });
    },
    onError: (error) => toast.error((error as Error).message || "Could not update job"),
  });

  const setStatus = useMutation({
    mutationFn: ({ id, status }: { id: string; status: ApplicationStatus }) =>
      applicationService.updateStatus(id, status),
    onSuccess: () => {
      toast.success("Application status updated");
      queryClient.invalidateQueries({ queryKey: ["recruiter", "applications"] });
    },
    onError: (error) => toast.error((error as Error).message || "Could not update status"),
  });

  return (
    <>
      <Button asChild variant="ghost" size="sm" className="-ml-2 w-fit">
        <Link to="/recruiter/jobs">
          <ArrowLeft className="mr-1.5 h-4 w-4" /> Back to postings
        </Link>
      </Button>

      <PageHeader
        title={job.data?.title ?? "Manage role"}
        description={job.data ? `${job.data.location} · ${job.data.applicantCount ?? 0} applicants` : undefined}
      />

      <Tabs defaultValue="ranking">
        <TabsList>
          <TabsTrigger value="ranking">AI ranking</TabsTrigger>
          <TabsTrigger value="applicants">Applicants</TabsTrigger>
          <TabsTrigger value="edit">Edit role</TabsTrigger>
        </TabsList>

        <TabsContent value="ranking" className="mt-6 space-y-4">
          <QueryBoundary isLoading={ranking.isLoading} error={ranking.error} onRetry={ranking.refetch}>
            {ranking.data?.length ? (
              ranking.data.map((candidate, index) => (
                <Card key={candidate.candidateId}>
                  <CardHeader>
                    <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-start gap-4">
                      <span className="grid h-8 w-8 shrink-0 place-items-center rounded-full bg-secondary font-display text-sm font-semibold">
                        {index + 1}
                      </span>
                      <div className="min-w-0">
                        <CardTitle className="truncate font-display text-base">
                          {candidate.candidateName}
                        </CardTitle>
                        <CardDescription className="truncate">{candidate.headline}</CardDescription>
                      </div>
                      <MatchScore score={candidate.matchScore} size={64} />
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div>
                      <p className="mb-1.5 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                        Why this ranking
                      </p>
                      <ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
                        {candidate.topReasons.map((reason) => (
                          <li key={reason}>{reason}</li>
                        ))}
                      </ul>
                    </div>
                    <div className="flex flex-wrap gap-1.5">
                      {candidate.matchedSkills.map((skill) => (
                        <Badge key={skill} className="bg-success/15 text-success hover:bg-success/15">
                          {skill}
                        </Badge>
                      ))}
                      {candidate.missingSkills.map((skill) => (
                        <Badge key={skill} variant="outline" className="border-destructive/40 text-destructive">
                          {skill}
                        </Badge>
                      ))}
                    </div>
                    <Button asChild variant="outline" size="sm">
                      <Link
                        to="/recruiter/candidates/$candidateId"
                        params={{ candidateId: candidate.candidateId }}
                      >
                        View candidate
                      </Link>
                    </Button>
                  </CardContent>
                </Card>
              ))
            ) : (
              <EmptyState icon={Users} title="No ranked candidates yet" />
            )}
          </QueryBoundary>
        </TabsContent>

        <TabsContent value="applicants" className="mt-6 space-y-4">
          <QueryBoundary isLoading={applications.isLoading} error={applications.error} onRetry={applications.refetch}>
            {applications.data?.length ? (
              applications.data.map((application) => (
                <Card key={application.id}>
                  <CardContent className="grid gap-4 p-5 sm:grid-cols-[auto_minmax(0,1fr)_auto] sm:items-center">
                    <MatchScore score={application.matchScore ?? 0} size={56} />
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <p className="truncate font-medium">{application.candidateName}</p>
                        <StatusBadge status={application.status} />
                      </div>
                      <Link
                        to="/recruiter/candidates/$candidateId"
                        params={{ candidateId: application.candidateId }}
                        className="text-sm text-muted-foreground underline-offset-2 hover:underline"
                      >
                        View profile & resume
                      </Link>
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
                          <SelectItem key={status} value={status} className="capitalize">
                            {status.replace("-", " ")}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </CardContent>
                </Card>
              ))
            ) : (
              <EmptyState icon={Users} title="No applicants for this role yet" />
            )}
          </QueryBoundary>
        </TabsContent>

        <TabsContent value="edit" className="mt-6">
          <QueryBoundary
            isLoading={job.isLoading}
            error={job.error}
            onRetry={job.refetch}
            skeleton={<Skeleton className="h-[32rem] w-full" />}
          >
            {job.data ? (
              <JobForm
                initial={job.data}
                submitLabel="Save changes"
                submitting={update.isPending}
                onSubmit={(payload) => update.mutate(payload)}
              />
            ) : null}
          </QueryBoundary>
        </TabsContent>
      </Tabs>
    </>
  );
}
