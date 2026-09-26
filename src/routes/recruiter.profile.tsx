import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { z } from "zod";

import { PageHeader } from "@/components/common/PageHeader";
import { QueryBoundary } from "@/components/common/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { recruiterService } from "@/services";
import type { RecruiterProfile } from "@/types";

const schema = z.object({
  fullName: z.string().trim().min(2, "Enter your full name").max(100),
  email: z.string().trim().email("Enter a valid email").max(255),
  companyName: z.string().trim().min(2, "Enter your company name").max(120),
  companyWebsite: z.string().trim().max(255).optional(),
  companySize: z.string().trim().max(50).optional(),
  industry: z.string().trim().max(100).optional(),
  location: z.string().trim().max(120).optional(),
  about: z.string().trim().max(1000).optional(),
});

export const Route = createFileRoute("/recruiter/profile")({
  head: () => ({
    meta: [
      { title: "Company profile — XHire" },
      { name: "description", content: "Manage your recruiter and company details." },
      { property: "og:title", content: "Company profile — XHire" },
      { property: "og:description", content: "Edit your XHire recruiter profile." },
    ],
  }),
  component: RecruiterProfilePage,
});

function RecruiterProfilePage() {
  const queryClient = useQueryClient();
  const profile = useQuery({
    queryKey: ["recruiter", "profile"],
    queryFn: () => recruiterService.getMyProfile(),
  });

  const [form, setForm] = useState<Partial<RecruiterProfile>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (profile.data) setForm(profile.data);
  }, [profile.data]);

  const save = useMutation({
    mutationFn: (payload: Partial<RecruiterProfile>) => recruiterService.updateMyProfile(payload),
    onSuccess: () => {
      toast.success("Profile saved");
      queryClient.invalidateQueries({ queryKey: ["recruiter", "profile"] });
    },
    onError: (error) => toast.error((error as Error).message || "Could not save profile"),
  });

  function set<K extends keyof RecruiterProfile>(key: K, value: RecruiterProfile[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    const parsed = schema.safeParse({
      fullName: form.fullName ?? "",
      email: form.email ?? "",
      companyName: form.companyName ?? "",
      companyWebsite: form.companyWebsite,
      companySize: form.companySize,
      industry: form.industry,
      location: form.location,
      about: form.about,
    });
    if (!parsed.success) {
      setErrors(Object.fromEntries(parsed.error.issues.map((i) => [String(i.path[0]), i.message])));
      return;
    }
    setErrors({});
    save.mutate(parsed.data);
  }

  return (
    <>
      <PageHeader title="Company profile" description="Candidates see this on every role you post." />

      <QueryBoundary
        isLoading={profile.isLoading}
        error={profile.error}
        onRetry={profile.refetch}
        skeleton={<Skeleton className="h-[28rem] w-full" />}
      >
        <form onSubmit={handleSubmit} className="space-y-6" noValidate>
          <Card>
            <CardHeader>
              <CardTitle className="font-display text-base">Recruiter</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="fullName">Full name</Label>
                <Input id="fullName" value={form.fullName ?? ""} onChange={(e) => set("fullName", e.target.value)} />
                {errors["fullName"] ? <p className="text-xs text-destructive">{errors["fullName"]}</p> : null}
              </div>
              <div className="space-y-2">
                <Label htmlFor="email">Email</Label>
                <Input id="email" type="email" value={form.email ?? ""} onChange={(e) => set("email", e.target.value)} />
                {errors["email"] ? <p className="text-xs text-destructive">{errors["email"]}</p> : null}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="font-display text-base">Company</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="companyName">Company name</Label>
                <Input
                  id="companyName"
                  value={form.companyName ?? ""}
                  onChange={(e) => set("companyName", e.target.value)}
                />
                {errors["companyName"] ? (
                  <p className="text-xs text-destructive">{errors["companyName"]}</p>
                ) : null}
              </div>
              <div className="space-y-2">
                <Label htmlFor="companyWebsite">Website</Label>
                <Input
                  id="companyWebsite"
                  value={form.companyWebsite ?? ""}
                  onChange={(e) => set("companyWebsite", e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="industry">Industry</Label>
                <Input id="industry" value={form.industry ?? ""} onChange={(e) => set("industry", e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="companySize">Company size</Label>
                <Input
                  id="companySize"
                  value={form.companySize ?? ""}
                  onChange={(e) => set("companySize", e.target.value)}
                  placeholder="e.g. 201-500"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="location">Location</Label>
                <Input id="location" value={form.location ?? ""} onChange={(e) => set("location", e.target.value)} />
              </div>
              <div className="space-y-2 sm:col-span-2">
                <Label htmlFor="about">About the company</Label>
                <Textarea
                  id="about"
                  rows={4}
                  maxLength={1000}
                  value={form.about ?? ""}
                  onChange={(e) => set("about", e.target.value)}
                />
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
