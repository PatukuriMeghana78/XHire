import { useState } from "react";
import { z } from "zod";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import type { Job, JobType, WorkMode } from "@/types";

const jobSchema = z.object({
  title: z.string().trim().min(3, "Enter a job title").max(120),
  location: z.string().trim().min(2, "Enter a location").max(120),
  description: z.string().trim().min(30, "Describe the role in at least 30 characters").max(5000),
  salaryMin: z.number().min(0).max(100000000).optional(),
  salaryMax: z.number().min(0).max(100000000).optional(),
  experienceMin: z.number().min(0).max(50).optional(),
});

export function JobForm({
  initial,
  submitLabel,
  submitting,
  onSubmit,
}: {
  initial?: Partial<Job>;
  submitLabel: string;
  submitting?: boolean;
  onSubmit: (payload: Partial<Job>) => void;
}) {
  const [form, setForm] = useState<Partial<Job>>({
    title: "",
    location: "",
    description: "",
    jobType: "full-time",
    workMode: "onsite",
    status: "open",
    ...initial,
  });
  const [skills, setSkills] = useState<string[]>(initial?.requiredSkills ?? []);
  const [skillDraft, setSkillDraft] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});

  function set<K extends keyof Job>(key: K, value: Job[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  function addSkill() {
    const value = skillDraft.trim();
    if (!value || skills.includes(value) || skills.length >= 20) return;
    setSkills((prev) => [...prev, value]);
    setSkillDraft("");
  }

  function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    const parsed = jobSchema.safeParse({
      title: form.title ?? "",
      location: form.location ?? "",
      description: form.description ?? "",
      salaryMin: form.salaryMin,
      salaryMax: form.salaryMax,
      experienceMin: form.experienceMin,
    });
    if (!parsed.success) {
      setErrors(Object.fromEntries(parsed.error.issues.map((i) => [String(i.path[0]), i.message])));
      return;
    }
    if (!skills.length) {
      setErrors({ skills: "Add at least one required skill" });
      return;
    }
    setErrors({});
    onSubmit({ ...form, ...parsed.data, requiredSkills: skills });
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-6" noValidate>
      <Card>
        <CardHeader>
          <CardTitle className="font-display text-base">Role details</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2 sm:col-span-2">
            <Label htmlFor="title">Job title</Label>
            <Input id="title" value={form.title ?? ""} onChange={(e) => set("title", e.target.value)} />
            {errors["title"] ? <p className="text-xs text-destructive">{errors["title"]}</p> : null}
          </div>
          <div className="space-y-2">
            <Label htmlFor="location">Location</Label>
            <Input id="location" value={form.location ?? ""} onChange={(e) => set("location", e.target.value)} />
            {errors["location"] ? <p className="text-xs text-destructive">{errors["location"]}</p> : null}
          </div>
          <div className="space-y-2">
            <Label>Work mode</Label>
            <Select value={form.workMode} onValueChange={(v) => set("workMode", v as WorkMode)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="onsite">Onsite</SelectItem>
                <SelectItem value="remote">Remote</SelectItem>
                <SelectItem value="hybrid">Hybrid</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>Job type</Label>
            <Select value={form.jobType} onValueChange={(v) => set("jobType", v as JobType)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="full-time">Full-time</SelectItem>
                <SelectItem value="part-time">Part-time</SelectItem>
                <SelectItem value="internship">Internship</SelectItem>
                <SelectItem value="contract">Contract</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>Status</Label>
            <Select value={form.status} onValueChange={(v) => set("status", v as Job["status"])}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="open">Open</SelectItem>
                <SelectItem value="draft">Draft</SelectItem>
                <SelectItem value="closed">Closed</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="salaryMin">Salary min (₹/yr)</Label>
            <Input
              id="salaryMin"
              type="number"
              min={0}
              value={form.salaryMin ?? ""}
              onChange={(e) => set("salaryMin", e.target.value ? Number(e.target.value) : undefined)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="salaryMax">Salary max (₹/yr)</Label>
            <Input
              id="salaryMax"
              type="number"
              min={0}
              value={form.salaryMax ?? ""}
              onChange={(e) => set("salaryMax", e.target.value ? Number(e.target.value) : undefined)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="experienceMin">Minimum experience (years)</Label>
            <Input
              id="experienceMin"
              type="number"
              min={0}
              max={50}
              value={form.experienceMin ?? ""}
              onChange={(e) => set("experienceMin", e.target.value ? Number(e.target.value) : undefined)}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="font-display text-base">Description & skills</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              rows={6}
              maxLength={5000}
              value={form.description ?? ""}
              onChange={(e) => set("description", e.target.value)}
            />
            {errors["description"] ? (
              <p className="text-xs text-destructive">{errors["description"]}</p>
            ) : null}
          </div>

          <div className="space-y-2">
            <Label>Required skills</Label>
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
            {errors["skills"] ? <p className="text-xs text-destructive">{errors["skills"]}</p> : null}
          </div>
        </CardContent>
      </Card>

      <div className="flex justify-end">
        <Button type="submit" disabled={submitting}>
          {submitting ? "Saving…" : submitLabel}
        </Button>
      </div>
    </form>
  );
}
