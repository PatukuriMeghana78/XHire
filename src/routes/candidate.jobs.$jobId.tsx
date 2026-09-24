import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, Briefcase, Clock, MapPin } from "lucide-react";
import { toast } from "sonner";

import { ExplanationPanel } from "@/components/ai/ExplanationPanel";
import { PageHeader } from "@/components/common/PageHeader";
import { QueryBoundary } from "@/components/common/QueryBoundary";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { formatRelativeDate, formatSalary } from "@/lib/format";
import { aiService, applicationService, jobService } from "@/services";

export const Route = createFileRoute("/candidate/jobs/$jobId")({
  head: () => ({
    meta: [
      { title: "Job details — XHire" },
      { name: "description", content: "Role details with your explainable AI match score." },
      { property: "og:title", content: "Job details — XHire" },
      { property: "og:description", content: "Role details and match explanation." },
    ],
  }),
  component: JobDetailPage,
});

function JobDetailPage() {
  const { jobId } = Route.useParams();
  const queryClient = useQueryClient();

  const job = useQuery({ queryKey: ["jobs", jobId], queryFn: () => jobService.getJob(jobId) });
  const match = useQuery({
    queryKey: ["ai", "match", jobId],
    queryFn: () => aiService.getJobMatchExplanation(jobId),
  });
  const applications = useQuery({
    queryKey: ["applications", "mine"],
    queryFn: () => applicationService.listMyApplications(),
  });

  const alreadyApplied = (applications.data ?? []).some((a) => a.jobId === jobId);

  const apply = useMutation({
    mutationFn: () => applicationService.applyToJob(jobId),
    onSuccess: () => {
      toast.success("Application submitted");
      queryClient.invalidateQueries({ queryKey: ["applications"] });
    },
    onError: (error) => toast.error((error as Error).message || "Could not apply"),
  });

  return (
    <>
      <Button asChild variant="ghost" size="sm" className="-ml-2 w-fit">
        <Link to="/candidate/jobs">
          <ArrowLeft className="mr-1.5 h-4 w-4" /> Back to jobs
        </Link>
      </Button>

      <QueryBoundary
        isLoading={job.isLoading}
        error={job.error}
        onRetry={job.refetch}
        skeleton={<Skeleton className="h-64 w-full" />}
      >
        {job.data ? (
          <>
            <PageHeader
              title={job.data.title}
              description={`${job.data.companyName} · ${job.data.location}`}
              actions={
                <Button
                  onClick={() => apply.mutate()}
                  disabled={alreadyApplied || apply.isPending}
                >
                  {alreadyApplied ? "Applied" : apply.isPending ? "Applying…" : "Apply now"}
                </Button>
              }
            />

            <div className="grid gap-6 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)]">
              <Card>
                <CardHeader>
                  <CardTitle className="font-display text-base">Role overview</CardTitle>
                </CardHeader>
                <CardContent className="space-y-5">
                  <div className="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-muted-foreground">
                    <span className="inline-flex items-center gap-1.5">
                      <MapPin className="h-4 w-4" aria-hidden /> {job.data.location}
                    </span>
                    <span className="inline-flex items-center gap-1.5 capitalize">
                      <Briefcase className="h-4 w-4" aria-hidden /> {job.data.jobType} · {job.data.workMode}
                    </span>
                    <span className="inline-flex items-center gap-1.5">
                      <Clock className="h-4 w-4" aria-hidden /> {formatRelativeDate(job.data.postedAt)}
                    </span>
                  </div>

                  <p className="text-sm leading-relaxed">{job.data.description}</p>

                  {job.data.responsibilities?.length ? (
                    <div>
                      <p className="mb-2 text-sm font-medium">Responsibilities</p>
                      <ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
                        {job.data.responsibilities.map((item) => (
                          <li key={item}>{item}</li>
                        ))}
                      </ul>
                    </div>
                  ) : null}

                  <div>
                    <p className="mb-2 text-sm font-medium">Required skills</p>
                    <div className="flex flex-wrap gap-1.5">
                      {job.data.requiredSkills.map((skill) => (
                        <Badge key={skill} variant="secondary">
                          {skill}
                        </Badge>
                      ))}
                    </div>
                  </div>

                  <p className="text-sm font-medium">
                    {formatSalary(job.data.salaryMin, job.data.salaryMax)}
                  </p>
                </CardContent>
              </Card>

              <QueryBoundary
                isLoading={match.isLoading}
                error={match.error}
                onRetry={match.refetch}
                skeleton={<Skeleton className="h-96 w-full" />}
              >
                {match.data ? <ExplanationPanel explanation={match.data} /> : null}
              </QueryBoundary>
            </div>
          </>
        ) : null}
      </QueryBoundary>
    </>
  );
}
