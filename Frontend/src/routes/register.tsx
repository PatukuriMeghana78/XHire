import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { toast } from "sonner";
import { z } from "zod";

import { AuthShell } from "@/routes/login";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { homeForRole, useAuth } from "@/lib/auth";
import type { UserRole } from "@/types";

const searchSchema = z.object({
  role: z.enum(["candidate", "recruiter"]).catch("candidate"),
});

const formSchema = z.object({
  fullName: z.string().trim().min(2, "Enter your full name").max(100),
  email: z.string().trim().email("Enter a valid email").max(255),
  password: z.string().min(6, "Password must be at least 6 characters").max(128),
});

export const Route = createFileRoute("/register")({
  validateSearch: searchSchema,
  head: () => ({
    meta: [
      { title: "Create your account — XHire" },
      {
        name: "description",
        content: "Register as a candidate or recruiter on XHire and start matching with explainable AI.",
      },
      { property: "og:title", content: "Create your account — XHire" },
      { property: "og:description", content: "Join XHire as a candidate or recruiter." },
    ],
  }),
  component: RegisterPage,
});

function RegisterPage() {
  const { role } = Route.useSearch();
  const navigate = useNavigate();
  const { register } = useAuth();
  const [activeRole, setActiveRole] = useState<UserRole>(role);
  const [form, setForm] = useState({ fullName: "", email: "", password: "" });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    const parsed = formSchema.safeParse(form);
    if (!parsed.success) {
      setErrors(
        Object.fromEntries(
          parsed.error.issues.map((issue) => [String(issue.path[0]), issue.message]),
        ),
      );
      return;
    }
    setErrors({});
    setSubmitting(true);
    try {
      const user = await register({ ...parsed.data, role: activeRole });
      toast.success("Account created");
      navigate({ to: homeForRole(user.role) });
    } catch (error) {
      toast.error((error as Error).message || "Registration failed");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AuthShell
      title="Create your XHire account"
      subtitle="Choose how you'll use XHire."
      footer={
        <>
          Already registered?{" "}
          <Link to="/login" search={{ role: activeRole }} className="font-medium underline">
            Sign in
          </Link>
        </>
      }
    >
      <Tabs value={activeRole} onValueChange={(value) => setActiveRole(value as UserRole)}>
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="candidate">Candidate</TabsTrigger>
          <TabsTrigger value="recruiter">Recruiter</TabsTrigger>
        </TabsList>
      </Tabs>

      <form onSubmit={handleSubmit} className="mt-6 space-y-4" noValidate>
        <div className="space-y-2">
          <Label htmlFor="fullName">Full name</Label>
          <Input
            id="fullName"
            value={form.fullName}
            onChange={(e) => setForm((f) => ({ ...f, fullName: e.target.value }))}
            placeholder="Aarav Sharma"
          />
          {errors["fullName"] ? (
            <p className="text-xs text-destructive">{errors["fullName"]}</p>
          ) : null}
        </div>
        <div className="space-y-2">
          <Label htmlFor="email">Email</Label>
          <Input
            id="email"
            type="email"
            autoComplete="email"
            value={form.email}
            onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
            placeholder="you@example.com"
          />
          {errors["email"] ? <p className="text-xs text-destructive">{errors["email"]}</p> : null}
        </div>
        <div className="space-y-2">
          <Label htmlFor="password">Password</Label>
          <Input
            id="password"
            type="password"
            autoComplete="new-password"
            value={form.password}
            onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
          />
          {errors["password"] ? (
            <p className="text-xs text-destructive">{errors["password"]}</p>
          ) : null}
        </div>
        <Button type="submit" className="w-full" disabled={submitting}>
          {submitting ? "Creating account…" : `Create ${activeRole} account`}
        </Button>
      </form>
    </AuthShell>
  );
}
