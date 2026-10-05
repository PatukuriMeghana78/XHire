import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { toast } from "sonner";
import { z } from "zod";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { homeForRole, useAuth } from "@/lib/auth";
import type { UserRole } from "@/types";

const searchSchema = z.object({
  role: z.enum(["candidate", "recruiter"]).catch("candidate"),
});

const formSchema = z.object({
  email: z.string().trim().email("Enter a valid email").max(255),
  password: z.string().min(6, "Password must be at least 6 characters").max(128),
});

export const Route = createFileRoute("/login")({
  validateSearch: searchSchema,
  head: () => ({
    meta: [
      { title: "Sign in — XHire" },
      { name: "description", content: "Sign in to your XHire candidate or recruiter account." },
      { property: "og:title", content: "Sign in — XHire" },
      { property: "og:description", content: "Access your XHire dashboard." },
    ],
  }),
  component: LoginPage,
});

function LoginPage() {
  const { role } = Route.useSearch();
  const navigate = useNavigate();
  const { login } = useAuth();
  const [activeRole, setActiveRole] = useState<UserRole>(role);
  const [form, setForm] = useState({ email: "", password: "" });
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
      const user = await login({ ...parsed.data, role: activeRole });
      toast.success(`Welcome back, ${user.fullName.split(" ")[0]}`);
      navigate({ to: homeForRole(user.role) });
    } catch (error) {
      toast.error((error as Error).message || "Sign in failed");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AuthShell
      title="Sign in to XHire"
      subtitle="Continue to your dashboard."
      footer={
        <>
          New to XHire?{" "}
          <Link to="/register" search={{ role: activeRole }} className="font-medium underline">
            Create an account
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
            autoComplete="current-password"
            value={form.password}
            onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
          />
          {errors["password"] ? (
            <p className="text-xs text-destructive">{errors["password"]}</p>
          ) : null}
        </div>
        <Button type="submit" className="w-full" disabled={submitting}>
          {submitting ? "Signing in…" : `Sign in as ${activeRole}`}
        </Button>
      </form>
    </AuthShell>
  );
}

export function AuthShell({
  title,
  subtitle,
  children,
  footer,
}: {
  title: string;
  subtitle: string;
  children: React.ReactNode;
  footer: React.ReactNode;
}) {
  return (
    <div className="relative flex min-h-screen items-center justify-center px-4 py-12">
      <div className="grid-bg absolute inset-0 opacity-50" aria-hidden />
      <Card className="relative w-full max-w-md">
        <CardHeader className="space-y-3">
          <Link to="/" className="flex items-center gap-2">
            <span className="grid h-8 w-8 place-items-center rounded-lg bg-primary font-display text-sm font-bold text-primary-foreground">
              X
            </span>
            <span className="font-display text-base font-semibold">XHire</span>
          </Link>
          <div>
            <CardTitle className="font-display text-xl">{title}</CardTitle>
            <CardDescription>{subtitle}</CardDescription>
          </div>
        </CardHeader>
        <CardContent>
          {children}
          <p className="mt-6 text-center text-sm text-muted-foreground">{footer}</p>
        </CardContent>
      </Card>
    </div>
  );
}
