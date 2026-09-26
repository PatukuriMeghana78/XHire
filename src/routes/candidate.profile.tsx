import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { z } from "zod";

import { PageHeader } from "@/components/common/PageHeader";
import { QueryBoundary } from "@/components/common/QueryBoundary";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { candidateService } from "@/services";
import type { CandidateProfile } from "@/types";

const profileSchema = z.object({
  fullName: z.string().trim().min(2, "Enter your full name").max(100),
  email: z.string().trim().email("Enter a valid email").max(255),
  phone: z.string().trim().max(20).optional(),
  headline: z.string().trim().max(140).optional(),
  location: z.string().trim().max(100).optional(),
  experienceYears: z.number().min(0).max(50).optional(),
  education: z.string().trim().max(200).optional(),
  about: z.string().trim().max(1000).optional(),
  linkedinUrl: z.string().trim().max(255).optional(),
  githubUrl: z.string().trim().max(255).optional(),
});

export const Route = createFileRoute("/candidate/profile")({
  head: () => ({
    meta: [
      { title: "My profile — XHire" },
      { name: "description", content: "Keep your candidate profile and skills up to date." },
      { property: "og:title", content: "My profile — XHire" },
      { property: "og:description", content: "Edit your XHire candidate profile." },
    ],
  }),
  component: CandidateProfilePage,
});

function CandidateProfilePage() {
  const queryClient = useQueryClient();
  const profile = useQuery({
    queryKey: ["candidate", "profile"],
    queryFn: () => candidateService.getMyProfile(),
  });

  const [form, setForm] = useState<Partial<CandidateProfile>>({});
  const [skills, setSkills] = useState<string[]>([]);
  const [skillDraft, setSkillDraft] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (profile.data) {
      setForm(profile.data);
      setSkills(profile.data.skills ?? []);
    }
  }, [profile.data]);

  const save = useMutation({
    mutationFn: (payload: Partial<CandidateProfile>) => candidateService.updateMyProfile(payload),
    onSuccess: () => {
      toast.success("Profile saved");
      queryClient.invalidateQueries({ queryKey: ["candidate", "profile"] });
    },
    onError: (error) => toast.error((error as Error).message || "Could not save profile"),
  });

  function set<K extends keyof CandidateProfile>(key: K, value: CandidateProfile[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  function addSkill() {
    const value = skillDraft.trim();
    if (!value || skills.includes(value) || skills.length >= 30) return;
    setSkills((prev) => [...prev, value]);
    setSkillDraft("");
  }

  function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    const parsed = profileSchema.safeParse({
      fullName: form.fullName ?? "",
      email: form.email ?? "",
      phone: form.phone,
      headline: form.headline,
      location: form.location,
      experienceYears: form.experienceYears,
      education: form.education,
      about: form.about,
      linkedinUrl: form.linkedinUrl,
      githubUrl: form.githubUrl,
    });
    if (!parsed.success) {
      setErrors(
        Object.fromEntries(parsed.error.issues.map((i) => [String(i.path[0]), i.message])),
      );
      return;
    }
    setErrors({});
    save.mutate({ ...parsed.data, skills });
  }

  return (
    <>
      <PageHeader
        title="My profile"
        description="Recruiters and the matching engine read this profile."
      />

      <QueryBoundary
        isLoading={profile.isLoading}
        error={profile.error}
        onRetry={profile.refetch}
        skeleton={<Skeleton className="h-[32rem] w-full" />}
      >
        <form onSubmit={handleSubmit} className="space-y-6" noValidate>
          <Card>
            <CardHeader>
              <CardTitle className="font-display text-base">Basics</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-4 sm:grid-cols-2">
              <Field label="Full name" error={errors["fullName"]}>
                <Input value={form.fullName ?? ""} onChange={(e) => set("fullName", e.target.value)} />
              </Field>
              <Field label="Email" error={errors["email"]}>
                <Input type="email" value={form.email ?? ""} onChange={(e) => set("email", e.target.value)} />
              </Field>
              <Field label="Phone">
                <Input value={form.phone ?? ""} onChange={(e) => set("phone", e.target.value)} />
              </Field>
              <Field label="Location">
                <Input value={form.location ?? ""} onChange={(e) => set("location", e.target.value)} />
              </Field>
              <Field label="Headline" error={errors["headline"]}>
                <Input value={form.headline ?? ""} onChange={(e) => set("headline", e.target.value)} />
              </Field>
              <Field label="Years of experience">
                <Input
                  type="number"
                  min={0}
                  max={50}
                  value={form.experienceYears ?? 0}
                  onChange={(e) => set("experienceYears", Number(e.target.value))}
                />
              </Field>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="font-display text-base">Skills</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex flex-wrap gap-1.5">
                {skills.map((skill) => (
                  <Badge
                    key={skill}
                    variant="secondary"
                    className="cursor-pointer"
                    onClick={() => setSkills((prev) => prev.filter((s) => s !== skill))}
                  >
                    {skill} ×
                  </Badge>
                ))}
                {skills.length === 0 ? (
                  <p className="text-sm text-muted-foreground">No skills added yet.</p>
                ) : null}
              </div>
              <div className="flex gap-2">
                <Input
                  value={skillDraft}
                  maxLength={40}
                  onChange={(e) => setSkillDraft(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault();
                      addSkill();
                    }
                  }}
                  placeholder="Add a skill and press Enter"
                />
                <Button type="button" variant="outline" onClick={addSkill}>
                  Add
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="font-display text-base">Background</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <Field label="Education">
                <Input value={form.education ?? ""} onChange={(e) => set("education", e.target.value)} />
              </Field>
              <Field label="About" error={errors["about"]}>
                <Textarea
                  rows={4}
                  maxLength={1000}
                  value={form.about ?? ""}
                  onChange={(e) => set("about", e.target.value)}
                />
              </Field>
              <div className="grid gap-4 sm:grid-cols-2">
                <Field label="LinkedIn URL">
                  <Input value={form.linkedinUrl ?? ""} onChange={(e) => set("linkedinUrl", e.target.value)} />
                </Field>
                <Field label="GitHub URL">
                  <Input value={form.githubUrl ?? ""} onChange={(e) => set("githubUrl", e.target.value)} />
                </Field>
              </div>
            </CardContent>
          </Card>

          <div className="flex justify-end">
            <Button type="submit" disabled={save.isPending}>
              {save.isPending ? "Saving…" : "Save profile"}
            </Button>
          </div>
        </form>
      </QueryBoundary>
    </>
  );
}

function Field({
  label,
  error,
  children,
}: {
  label: string;
  error?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      {children}
      {error ? <p className="text-xs text-destructive">{error}</p> : null}
    </div>
  );
}
