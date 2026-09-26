import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Briefcase, Pencil, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { EmptyState } from "@/components/common/EmptyState";
import { PageHeader } from "@/components/common/PageHeader";
import { QueryBoundary } from "@/components/common/QueryBoundary";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { formatRelativeDate, formatSalary } from "@/lib/format";
import { jobService } from "@/services";

export const Route = createFileRoute("/recruiter/jobs/")({
  head: () => ({
    meta: [
      { title: "Job postings — XHire" },
      { name: "description", content: "Create, edit and close your job postings." },
      { property: "og:title", content: "Job postings — XHire" },
      { property: "og:description", content: "Manage your XHire job postings." },
    ],
  }),
  component: RecruiterJobsPage,
});

function RecruiterJobsPage() {
  const queryClient = useQueryClient();
  const jobs = useQuery({ queryKey: ["recruiter", "jobs"], queryFn: () => jobService.listRecruiterJobs() });

  const remove = useMutation({
    mutationFn: (id: string) => jobService.deleteJob(id),
    onSuccess: () => {
      toast.success("Job deleted");
      queryClient.invalidateQueries({ queryKey: ["recruiter", "jobs"] });
    },
    onError: (error) => toast.error((error as Error).message || "Could not delete job"),
  });

  return (
    <>
      <PageHeader
        title="Job postings"
        description="Everything you've published, with applicant counts."
        actions={
          <Button asChild>
            <Link to="/recruiter/jobs/new">
              <Plus className="mr-2 h-4 w-4" /> Post a job
            </Link>
          </Button>
        }
      />

      <QueryBoundary isLoading={jobs.isLoading} error={jobs.error} onRetry={jobs.refetch}>
        {jobs.data?.length ? (
          <div className="space-y-4">
            {jobs.data.map((job) => (
              <Card key={job.id}>
                <CardContent className="grid gap-4 p-5 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
                  <div className="min-w-0 space-y-2">
                    <div className="flex flex-wrap items-center gap-2">
                      <Link
                        to="/recruiter/jobs/$jobId"
                        params={{ jobId: job.id }}
                        className="truncate font-medium hover:underline"
                      >
                        {job.title}
                      </Link>
                      <Badge variant={job.status === "open" ? "secondary" : "outline"} className="capitalize">
                        {job.status}
                      </Badge>
                    </div>
                    <p className="text-sm text-muted-foreground">
                      {job.location} · {job.jobType} · {formatSalary(job.salaryMin, job.salaryMax)}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {job.applicantCount ?? 0} applicants · posted {formatRelativeDate(job.postedAt)}
                    </p>
                  </div>
                  <div className="flex shrink-0 gap-2">
                    <Button asChild variant="outline" size="sm">
                      <Link to="/recruiter/jobs/$jobId" params={{ jobId: job.id }}>
                        <Pencil className="mr-1.5 h-3.5 w-3.5" /> Manage
                      </Link>
                    </Button>
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button variant="ghost" size="sm" aria-label={`Delete ${job.title}`}>
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Delete this job posting?</AlertDialogTitle>
                          <AlertDialogDescription>
                            "{job.title}" and its applicant links will be removed. This can't be undone.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>Cancel</AlertDialogCancel>
                          <AlertDialogAction onClick={() => remove.mutate(job.id)}>Delete</AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        ) : (
          <EmptyState
            icon={Briefcase}
            title="No job postings yet"
            description="Publish a role to start receiving ranked applicants."
            action={
              <Button asChild size="sm">
                <Link to="/recruiter/jobs/new">Post a job</Link>
              </Button>
            }
          />
        )}
      </QueryBoundary>
    </>
  );
}
