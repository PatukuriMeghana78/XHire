import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { Download, FileUp, Upload } from "lucide-react";
import { useRef } from "react";
import { toast } from "sonner";

import { SkillGapChart } from "@/components/ai/SkillGapChart";
import { EmptyState } from "@/components/common/EmptyState";
import { MatchScore } from "@/components/common/MatchScore";
import { PageHeader } from "@/components/common/PageHeader";
import { QueryBoundary } from "@/components/common/QueryBoundary";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import { formatDate } from "@/lib/format";
import { aiService, resumeService } from "@/services";

export const Route = createFileRoute("/candidate/resume")({
  head: () => ({
    meta: [
      { title: "Resume & AI insight — XHire" },
      {
        name: "description",
        content: "Upload your resume and review AI analysis, extracted skills and skill gaps.",
      },
      { property: "og:title", content: "Resume & AI insight — XHire" },
      { property: "og:description", content: "Resume analysis and skill-gap insight." },
    ],
  }),
  component: ResumePage,
});

function ResumePage() {
  const queryClient = useQueryClient();
  const fileInput = useRef<HTMLInputElement>(null);

  const resume = useQuery({ queryKey: ["resume", "mine"], queryFn: () => resumeService.getMyResume() });
  const analysis = useQuery({
    queryKey: ["resume", "analysis", resume.data?.id],
    queryFn: () => resumeService.getAnalysis(resume.data!.id),
    enabled: Boolean(resume.data?.id),
  });
  const skillGaps = useQuery({ queryKey: ["ai", "skill-gaps"], queryFn: () => aiService.getSkillGaps() });

  const upload = useMutation({
    mutationFn: (file: File) => resumeService.uploadResume(file),
    onSuccess: () => {
      toast.success("Resume uploaded");
      queryClient.invalidateQueries({ queryKey: ["resume"] });
      queryClient.invalidateQueries({ queryKey: ["ai"] });
    },
    onError: (error) => toast.error((error as Error).message || "Upload failed"),
  });

  function handleFile(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    if (file.size > 5 * 1024 * 1024) {
      toast.error("Resume must be smaller than 5 MB");
      return;
    }
    upload.mutate(file);
    event.target.value = "";
  }

  return (
    <>
      <PageHeader
        title="Resume & AI insight"
        description="Your resume drives every match score and skill-gap report."
        actions={
          <Button onClick={() => fileInput.current?.click()} disabled={upload.isPending}>
            <Upload className="mr-2 h-4 w-4" />
            {upload.isPending ? "Uploading…" : "Upload resume"}
          </Button>
        }
      />
      <input
        ref={fileInput}
        type="file"
        accept=".pdf,.doc,.docx"
        className="hidden"
        onChange={handleFile}
      />

      <QueryBoundary
        isLoading={resume.isLoading}
        error={resume.error}
        onRetry={resume.refetch}
        skeleton={<Skeleton className="h-28 w-full" />}
      >
        {resume.data ? (
          <Card>
            <CardContent className="flex flex-wrap items-center justify-between gap-4 p-5">
              <div className="flex min-w-0 items-center gap-3">
                <span className="grid h-10 w-10 shrink-0 place-items-center rounded-lg bg-secondary">
                  <FileUp className="h-5 w-5" aria-hidden />
                </span>
                <div className="min-w-0">
                  <p className="truncate font-medium">{resume.data.fileName}</p>
                  <p className="text-xs text-muted-foreground">
                    Uploaded {formatDate(resume.data.uploadedAt)}
                    {resume.data.sizeKb ? ` · ${resume.data.sizeKb} KB` : ""}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() =>
                    resume.data &&
                    resumeService
                      .downloadResume(resume.data.id, resume.data.fileName)
                      .catch((error) =>
                        toast.error((error as Error).message || "Download failed"),
                      )
                  }
                >
                  <Download className="mr-1.5 h-3.5 w-3.5" /> Download
                </Button>
                <Button variant="outline" size="sm" onClick={() => fileInput.current?.click()}>
                  Replace
                </Button>
              </div>
            </CardContent>
          </Card>
        ) : (
          <EmptyState
            icon={FileUp}
            title="No resume uploaded"
            description="Upload a PDF or Word file to unlock analysis and matching."
            action={<Button onClick={() => fileInput.current?.click()}>Upload resume</Button>}
          />
        )}
      </QueryBoundary>

      {resume.data ? (
        <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
          <QueryBoundary
            isLoading={analysis.isLoading}
            error={analysis.error}
            onRetry={analysis.refetch}
            skeleton={<Skeleton className="h-96 w-full" />}
          >
            {analysis.data ? (
              <Card>
                <CardHeader>
                  <div className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-4">
                    <div className="min-w-0">
                      <CardTitle className="font-display text-base">Resume analysis</CardTitle>
                      <CardDescription>Produced by the XHire analysis service.</CardDescription>
                    </div>
                    {analysis.data.parsed === false ? null : (
                      <MatchScore score={analysis.data.overallScore} label="score" />
                    )}
                  </div>
                </CardHeader>
                {analysis.data.parsed === false ? (
                  <CardContent className="space-y-5">
                    <div className="rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
                      <p className="font-medium text-foreground">Analysis unavailable</p>
                      <p className="mt-1">
                        {analysis.data.reason ??
                          "Resume content parsing is not available for this file."}
                      </p>
                    </div>
                    {(analysis.data.declaredSkills?.length ?? 0) > 0 ? (
                      <div>
                        <p className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                          Skills from your profile
                        </p>
                        <div className="flex flex-wrap gap-1.5">
                          {analysis.data.declaredSkills?.map((skill) => (
                            <Badge key={skill} variant="secondary">
                              {skill}
                            </Badge>
                          ))}
                        </div>
                      </div>
                    ) : null}
                  </CardContent>
                ) : (
                  <CardContent className="space-y-5">
                    {analysis.data.sections.map((section) => (
                      <div key={section.name} className="space-y-1.5">
                        <div className="flex items-baseline justify-between gap-3 text-sm">
                          <span className="font-medium">{section.name}</span>
                          <span className="text-muted-foreground">{section.score}/100</span>
                        </div>
                        <Progress value={section.score} className="h-1.5" />
                        <p className="text-xs text-muted-foreground">{section.feedback}</p>
                      </div>
                    ))}

                    <div>
                      <p className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                        Extracted skills
                      </p>
                      <div className="flex flex-wrap gap-1.5">
                        {analysis.data.extractedSkills.map((skill) => (
                          <Badge key={skill} variant="secondary">
                            {skill}
                          </Badge>
                        ))}
                      </div>
                    </div>

                    <div className="grid gap-4 sm:grid-cols-2">
                      <div>
                        <p className="mb-1.5 text-sm font-medium">Strengths</p>
                        <ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
                          {analysis.data.strengths.map((item) => (
                            <li key={item}>{item}</li>
                          ))}
                        </ul>
                      </div>
                      <div>
                        <p className="mb-1.5 text-sm font-medium">Improvements</p>
                        <ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
                          {analysis.data.improvements.map((item) => (
                            <li key={item}>{item}</li>
                          ))}
                        </ul>
                      </div>
                    </div>
                  </CardContent>
                )}
              </Card>
            ) : null}
          </QueryBoundary>

          <QueryBoundary
            isLoading={skillGaps.isLoading}
            error={skillGaps.error}
            onRetry={skillGaps.refetch}
            skeleton={<Skeleton className="h-96 w-full" />}
          >
            {skillGaps.data ? (
              <Card>
                <CardHeader>
                  <CardTitle className="font-display text-base">Skill gaps</CardTitle>
                  <CardDescription>
                    Your level against what your target roles ask for.
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <SkillGapChart gaps={skillGaps.data} />
                  <ul className="space-y-2">
                    {skillGaps.data
                      .filter((gap) => gap.priority !== "low")
                      .map((gap) => (
                        <li
                          key={gap.skill}
                          className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2 text-sm"
                        >
                          <span className="truncate font-medium">{gap.skill}</span>
                          <span className="shrink-0 text-xs text-muted-foreground">
                            {gap.current} → {gap.required} ·{" "}
                            <span className="capitalize">{gap.priority} priority</span>
                          </span>
                        </li>
                      ))}
                  </ul>
                </CardContent>
              </Card>
            ) : null}
          </QueryBoundary>
        </div>
      ) : null}
    </>
  );
}
