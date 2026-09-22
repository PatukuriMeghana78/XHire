import { cn } from "@/lib/utils";
import type { ApplicationStatus } from "@/types";

const LABELS: Record<ApplicationStatus, string> = {
  applied: "Applied",
  "under-review": "Under review",
  shortlisted: "Shortlisted",
  interview: "Interview",
  rejected: "Rejected",
  hired: "Hired",
};

const STYLES: Record<ApplicationStatus, string> = {
  applied: "bg-muted text-muted-foreground",
  "under-review": "bg-accent/20 text-accent-foreground",
  shortlisted: "bg-success/15 text-success",
  interview: "bg-primary/10 text-primary",
  rejected: "bg-destructive/10 text-destructive",
  hired: "bg-success text-success-foreground",
};

export function StatusBadge({
  status,
  className,
}: {
  status: ApplicationStatus;
  className?: string;
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
        STYLES[status],
        className,
      )}
    >
      {LABELS[status]}
    </span>
  );
}

export const applicationStatusLabel = (status: ApplicationStatus) => LABELS[status];
