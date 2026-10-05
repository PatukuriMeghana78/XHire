import { Link } from "@tanstack/react-router";
import { Briefcase, MapPin, Clock } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { formatRelativeDate, formatSalary } from "@/lib/format";
import type { Job } from "@/types";

export function JobCard({ job }: { job: Job }) {
  return (
    <Link to="/candidate/jobs/$jobId" params={{ jobId: job.id }} className="group block">
      <Card className="h-full transition-all group-hover:-translate-y-0.5 group-hover:border-primary/40 group-hover:shadow-md">
        <CardContent className="space-y-4 p-5">
          <div className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-3">
            <div className="min-w-0">
              <h3 className="truncate font-display text-base font-semibold">{job.title}</h3>
              <p className="truncate text-sm text-muted-foreground">{job.companyName}</p>
            </div>
            <Badge variant="secondary" className="shrink-0 capitalize">
              {job.workMode}
            </Badge>
          </div>

          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
            <span className="inline-flex items-center gap-1">
              <MapPin className="h-3.5 w-3.5" aria-hidden /> {job.location}
            </span>
            <span className="inline-flex items-center gap-1 capitalize">
              <Briefcase className="h-3.5 w-3.5" aria-hidden /> {job.jobType}
            </span>
            <span className="inline-flex items-center gap-1">
              <Clock className="h-3.5 w-3.5" aria-hidden /> {formatRelativeDate(job.postedAt)}
            </span>
          </div>

          <div className="flex flex-wrap gap-1.5">
            {job.requiredSkills.slice(0, 4).map((skill) => (
              <Badge key={skill} variant="outline" className="font-normal">
                {skill}
              </Badge>
            ))}
            {job.requiredSkills.length > 4 ? (
              <Badge variant="outline" className="font-normal">
                +{job.requiredSkills.length - 4}
              </Badge>
            ) : null}
          </div>

          <p className="text-sm font-medium">{formatSalary(job.salaryMin, job.salaryMax)}</p>
        </CardContent>
      </Card>
    </Link>
  );
}
