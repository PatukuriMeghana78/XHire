import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { toast } from "sonner";

import { PageHeader } from "@/components/common/PageHeader";
import { JobForm } from "@/components/jobs/JobForm";
import { Button } from "@/components/ui/button";
import { jobService } from "@/services";
import type { Job } from "@/types";

export const Route = createFileRoute("/recruiter/jobs/new")({
  head: () => ({
    meta: [
      { title: "Post a job — XHire" },
      { name: "description", content: "Publish a new role and start receiving ranked applicants." },
      { property: "og:title", content: "Post a job — XHire" },
      { property: "og:description", content: "Publish a new role on XHire." },
    ],
  }),
  component: NewJobPage,
});

function NewJobPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const create = useMutation({
    mutationFn: (payload: Partial<Job>) => jobService.createJob(payload),
    onSuccess: () => {
      toast.success("Job posted");
      queryClient.invalidateQueries({ queryKey: ["recruiter", "jobs"] });
      navigate({ to: "/recruiter/jobs" });
    },
    onError: (error) => toast.error((error as Error).message || "Could not post job"),
  });

  return (
    <>
      <Button asChild variant="ghost" size="sm" className="-ml-2 w-fit">
        <Link to="/recruiter/jobs">
          <ArrowLeft className="mr-1.5 h-4 w-4" /> Back to postings
        </Link>
      </Button>
      <PageHeader title="Post a job" description="Describe the role; skills drive candidate matching." />
      <JobForm submitLabel="Publish job" submitting={create.isPending} onSubmit={(p) => create.mutate(p)} />
    </>
  );
}
