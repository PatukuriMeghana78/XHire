import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { MatchScore } from "@/components/common/MatchScore";
import { Progress } from "@/components/ui/progress";
import type { MatchExplanation } from "@/types";

export function ExplanationPanel({ explanation }: { explanation: MatchExplanation }) {
  return (
    <Card>
      <CardHeader>
        <div className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-4">
          <div className="min-w-0">
            <CardTitle className="font-display">Why this match</CardTitle>
            <CardDescription className="mt-1">{explanation.summary}</CardDescription>
          </div>
          <MatchScore score={explanation.matchScore} />
        </div>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="space-y-4">
          {explanation.factors.map((factor) => (
            <div key={factor.label} className="space-y-1.5">
              <div className="flex items-baseline justify-between gap-3 text-sm">
                <span className="font-medium">{factor.label}</span>
                <span className="shrink-0 text-muted-foreground">
                  +{factor.contribution} pts · weight {Math.round(factor.weight * 100)}%
                </span>
              </div>
              <Progress value={(factor.contribution / (factor.weight * 100)) * 100} className="h-1.5" />
              <p className="text-xs text-muted-foreground">{factor.detail}</p>
            </div>
          ))}
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <p className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Matched skills
            </p>
            <div className="flex flex-wrap gap-1.5">
              {explanation.matchedSkills.map((skill) => (
                <Badge key={skill} className="bg-success/15 text-success hover:bg-success/15">
                  {skill}
                </Badge>
              ))}
            </div>
          </div>
          <div>
            <p className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Missing skills
            </p>
            <div className="flex flex-wrap gap-1.5">
              {explanation.missingSkills.length ? (
                explanation.missingSkills.map((skill) => (
                  <Badge key={skill} variant="outline" className="border-destructive/40 text-destructive">
                    {skill}
                  </Badge>
                ))
              ) : (
                <span className="text-sm text-muted-foreground">None — full coverage.</span>
              )}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
