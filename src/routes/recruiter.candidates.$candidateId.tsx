import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, Download, FileText, Mail, MapPin } from "lucide-react";

import { PageHeader } from "@/components/common/PageHeader";
import { QueryBoundary } from "@/components/common/QueryBoundary";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { formatDate } from "@/lib/format";
import { candidateService, resumeService } from "@/services";

export const Route = createFileRoute("/recruiter/candidates/$candidateId")({
  head: () => ({
    meta: [
      { title: "Candidate profile — XHire" },
      { name: "description", content: "Full candidate profile, skills and resume." },
      { property: "og:title", content: "Candidate profile — XHire" },
      { property: "og:description", content: "Review a candidate's profile and resume." },
    ],
  }),
  component: CandidateDetailPage,
});

function CandidateDetailPage() {
  const { candidateId } = Route.useParams();

  const candidate = useQuery({
    queryKey: ["candidate", candidateId],
    queryFn: () => candidateService.getCandidate(candidateId),
  });
  const resume = useQuery({
    queryKey: ["resume", "candidate", candidateId],
    queryFn: () => resumeService.getCandidateResume(candidateId),
  });

  return (
    <>
      <Button asChild variant="ghost" size="sm" className="-ml-2 w-fit">
        <Link to="/recruiter/applications">
          <ArrowLeft className="mr-1.5 h-4 w-4" /> Back to applications
        </Link>
      </Button>

      <QueryBoundary
        isLoading={candidate.isLoading}
        error={candidate.error}
        onRetry={candidate.refetch}
        skeleton={<Skeleton className="h-96 w-full" />}
      >
        {candidate.data ? (
          <>
            <PageHeader
              title={candidate.data.fullName}
              description={candidate.data.headline}
            />

            <div className="grid gap-6 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)]">
              <Card>
                <CardHeader>
                  <CardTitle className="font-display text-base">Profile</CardTitle>
                </CardHeader>
                <CardContent className="space-y-5">
                  <div className="flex flex-wrap gap-x-5 gap-y-2 text-sm text-muted-foreground">
                    <span className="inline-flex items-center gap-1.5">
                      <Mail className="h-4 w-4" aria-hidden /> {candidate.data.email}
                    </span>
                    {candidate.data.location ? (
                      <span className="inline-flex items-center gap-1.5">
                        <MapPin className="h-4 w-4" aria-hidden /> {candidate.data.location}
                      </span>
                    ) : null}
                  </div>

                  {candidate.data.about ? (
                    <p className="text-sm leading-relaxed">{candidate.data.about}</p>
                  ) : null}

                  <div className="grid gap-4 sm:grid-cols-2 text-sm">
                    <div>
                      <p className="text-xs uppercase tracking-wide text-muted-foreground">Experience</p>
                      <p className="mt-0.5">{candidate.data.experienceYears ?? 0} years</p>
                    </div>
                    <div>
                      <p className="text-xs uppercase tracking-wide text-muted-foreground">Education</p>
                      <p className="mt-0.5">{candidate.data.education ?? "—"}</p>
                    </div>
                  </div>

                  <div>
                    <p className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                      Skills
                    </p>
                    <div className="flex flex-wrap gap-1.5">
                      {candidate.data.skills.map((skill) => (
                        <Badge key={skill} variant="secondary">
                          {skill}
                        </Badge>
                      ))}
                    </div>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="font-display text-base">Resume</CardTitle>
                </CardHeader>
                <CardContent>
                  <QueryBoundary
                    isLoading={resume.isLoading}
                    error={resume.error}
                    onRetry={resume.refetch}
                    skeleton={<Skeleton className="h-24 w-full" />}
                  >
                    {resume.data ? (
                      <div className="flex items-center justify-between gap-3 rounded-lg border p-4">
                        <div className="flex min-w-0 items-center gap-3">
                          <FileText className="h-5 w-5 shrink-0 text-muted-foreground" aria-hidden />
                          <div className="min-w-0">
                            <p className="truncate text-sm font-medium">{resume.data.fileName}</p>
                            <p className="text-xs text-muted-foreground">
                              Uploaded {formatDate(resume.data.uploadedAt)}
                            </p>
                          </div>
                        </div>
                        <Button
                          asChild={Boolean(resume.data.fileUrl)}
                          variant="outline"
                          size="sm"
                          disabled={!resume.data.fileUrl}
                        >
                          {resume.data.fileUrl ? (
                            <a href={resume.data.fileUrl} target="_blank" rel="noreferrer">
                              <Download className="mr-1.5 h-3.5 w-3.5" /> Open
                            </a>
                          ) : (
                            <span>
                              <Download className="mr-1.5 h-3.5 w-3.5" /> Open
                            </span>
                          )}
                        </Button>
                      </div>
                    ) : (
                      <p className="text-sm text-muted-foreground">No resume on file.</p>
                    )}
                  </QueryBoundary>
                </CardContent>
              </Card>
            </div>
          </>
        ) : null}
      </QueryBoundary>
    </>
  );
}
