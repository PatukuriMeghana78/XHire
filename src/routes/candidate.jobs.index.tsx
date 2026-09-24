import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { Search, SearchX } from "lucide-react";
import { useState } from "react";

import { EmptyState } from "@/components/common/EmptyState";
import { PageHeader } from "@/components/common/PageHeader";
import { QueryBoundary } from "@/components/common/QueryBoundary";
import { JobCard } from "@/components/jobs/JobCard";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { jobService } from "@/services";
import type { JobFilters, JobType, WorkMode } from "@/types";

export const Route = createFileRoute("/candidate/jobs/")({
  head: () => ({
    meta: [
      { title: "Find jobs — XHire" },
      { name: "description", content: "Search and filter open roles matched to your profile." },
      { property: "og:title", content: "Find jobs — XHire" },
      { property: "og:description", content: "Search open roles on XHire." },
    ],
  }),
  component: JobSearchPage,
});

function JobSearchPage() {
  const [filters, setFilters] = useState<JobFilters>({
    search: "",
    location: "",
    jobType: "all",
    workMode: "all",
  });

  const jobs = useQuery({
    queryKey: ["jobs", filters],
    queryFn: () => jobService.listJobs(filters),
  });

  return (
    <>
      <PageHeader title="Find jobs" description="Filter open roles and open one to see your match." />

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <div className="relative sm:col-span-2">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
          <Input
            value={filters.search ?? ""}
            onChange={(e) => setFilters((f) => ({ ...f, search: e.target.value }))}
            placeholder="Search title, company or skill"
            className="pl-9"
            aria-label="Search jobs"
          />
        </div>
        <Input
          value={filters.location ?? ""}
          onChange={(e) => setFilters((f) => ({ ...f, location: e.target.value }))}
          placeholder="Location"
          aria-label="Filter by location"
        />
        <div className="grid grid-cols-2 gap-3">
          <Select
            value={filters.jobType ?? "all"}
            onValueChange={(value) => setFilters((f) => ({ ...f, jobType: value as JobType | "all" }))}
          >
            <SelectTrigger aria-label="Job type">
              <SelectValue placeholder="Type" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All types</SelectItem>
              <SelectItem value="full-time">Full-time</SelectItem>
              <SelectItem value="part-time">Part-time</SelectItem>
              <SelectItem value="internship">Internship</SelectItem>
              <SelectItem value="contract">Contract</SelectItem>
            </SelectContent>
          </Select>
          <Select
            value={filters.workMode ?? "all"}
            onValueChange={(value) => setFilters((f) => ({ ...f, workMode: value as WorkMode | "all" }))}
          >
            <SelectTrigger aria-label="Work mode">
              <SelectValue placeholder="Mode" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Any mode</SelectItem>
              <SelectItem value="onsite">Onsite</SelectItem>
              <SelectItem value="remote">Remote</SelectItem>
              <SelectItem value="hybrid">Hybrid</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <QueryBoundary
        isLoading={jobs.isLoading}
        error={jobs.error}
        onRetry={jobs.refetch}
        skeleton={
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {[0, 1, 2, 3, 4, 5].map((i) => (
              <Skeleton key={i} className="h-52 w-full" />
            ))}
          </div>
        }
      >
        {jobs.data?.length ? (
          <>
            <p className="text-sm text-muted-foreground">{jobs.data.length} roles found</p>
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              {jobs.data.map((job) => (
                <JobCard key={job.id} job={job} />
              ))}
            </div>
          </>
        ) : (
          <EmptyState
            icon={SearchX}
            title="No roles match these filters"
            description="Try a broader search term or clear the location filter."
          />
        )}
      </QueryBoundary>
    </>
  );
}
