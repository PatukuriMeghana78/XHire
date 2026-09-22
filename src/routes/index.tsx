import { createFileRoute, Link } from "@tanstack/react-router";
import { motion } from "motion/react";
import {
  ArrowRight,
  BarChart3,
  Brain,
  FileSearch,
  Gauge,
  ListChecks,
  ShieldCheck,
  Users,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

export const Route = createFileRoute("/")({
  head: () => ({
    meta: [
      { title: "XHire — Explainable AI Recruitment & Job Application Platform" },
      {
        name: "description",
        content:
          "XHire pairs candidates and recruiters with explainable AI: match scores you can audit, resume analysis, skill-gap insight and ranked shortlists.",
      },
      { property: "og:title", content: "XHire — Explainable AI Recruitment Platform" },
      {
        property: "og:description",
        content:
          "Match scores you can audit, resume analysis and ranked shortlists for recruiters.",
      },
    ],
  }),
  component: Landing,
});

const CANDIDATE_FEATURES = [
  { icon: FileSearch, title: "Resume analysis", body: "Section-by-section scoring with parser-friendly fixes." },
  { icon: Gauge, title: "Match scores", body: "See how strongly you fit a role before you apply." },
  { icon: ListChecks, title: "Skill gaps", body: "Know exactly which skills to close, ranked by impact." },
];

const RECRUITER_FEATURES = [
  { icon: Users, title: "Ranked shortlists", body: "Applicants ordered by evidence, not keyword luck." },
  { icon: Brain, title: "Reasoned rankings", body: "Every score breaks down into weighted factors." },
  { icon: BarChart3, title: "Pipeline view", body: "Track applications from applied to hired." },
];

function Landing() {
  return (
    <div className="min-h-screen bg-background">
      <header className="sticky top-0 z-40 border-b bg-background/85 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-3 sm:px-6">
          <Link to="/" className="flex items-center gap-2">
            <span className="grid h-8 w-8 place-items-center rounded-lg bg-primary font-display text-sm font-bold text-primary-foreground">
              X
            </span>
            <span className="font-display text-lg font-semibold">XHire</span>
          </Link>
          <div className="flex items-center gap-2">
            <Button asChild variant="ghost" size="sm">
              <Link to="/login">Sign in</Link>
            </Button>
            <Button asChild size="sm">
              <Link to="/register">Get started</Link>
            </Button>
          </div>
        </div>
      </header>

      <section className="relative overflow-hidden border-b">
        <div className="grid-bg absolute inset-0 opacity-60" aria-hidden />
        <div className="relative mx-auto max-w-6xl px-4 py-20 sm:px-6 lg:py-28">
          <motion.div
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
            className="max-w-2xl"
          >
            <Badge variant="secondary" className="mb-5">
              Explainable AI recruitment
            </Badge>
            <h1 className="text-4xl font-semibold leading-[1.05] sm:text-6xl">
              Hiring decisions that can{" "}
              <span className="bg-accent/50 px-1.5 [box-decoration-break:clone]">explain themselves</span>
            </h1>
            <p className="mt-6 max-w-xl text-lg text-muted-foreground">
              XHire scores candidate–job fit and shows the reasoning behind every number — matched
              skills, missing skills and the weight of each factor.
            </p>
            <div className="mt-8 flex flex-wrap gap-3">
              <Button asChild size="lg">
                <Link to="/register" search={{ role: "candidate" as const }}>
                  I'm looking for a job <ArrowRight className="ml-2 h-4 w-4" />
                </Link>
              </Button>
              <Button asChild size="lg" variant="outline">
                <Link to="/register" search={{ role: "recruiter" as const }}>
                  I'm hiring
                </Link>
              </Button>
            </div>
            <p className="mt-6 inline-flex items-center gap-2 text-xs text-muted-foreground">
              <ShieldCheck className="h-4 w-4" aria-hidden />
              Every score is auditable — no black-box decisions.
            </p>
          </motion.div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-4 py-16 sm:px-6 lg:py-24">
        <div className="grid gap-12 lg:grid-cols-2">
          {[
            { title: "For candidates", items: CANDIDATE_FEATURES },
            { title: "For recruiters", items: RECRUITER_FEATURES },
          ].map((group) => (
            <div key={group.title}>
              <h2 className="text-xl font-semibold">{group.title}</h2>
              <div className="mt-5 space-y-3">
                {group.items.map(({ icon: Icon, title, body }) => (
                  <Card key={title}>
                    <CardContent className="flex items-start gap-4 p-5">
                      <span className="grid h-10 w-10 shrink-0 place-items-center rounded-lg bg-secondary text-secondary-foreground">
                        <Icon className="h-5 w-5" aria-hidden />
                      </span>
                      <div className="min-w-0">
                        <p className="font-medium">{title}</p>
                        <p className="mt-0.5 text-sm text-muted-foreground">{body}</p>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="border-t bg-secondary/40">
        <div className="mx-auto flex max-w-6xl flex-col items-start gap-4 px-4 py-14 sm:px-6 md:flex-row md:items-center md:justify-between">
          <div>
            <h2 className="text-2xl font-semibold">Ready to see your match score?</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              Create an account as a candidate or a recruiter in under a minute.
            </p>
          </div>
          <Button asChild size="lg">
            <Link to="/register">Create your account</Link>
          </Button>
        </div>
      </section>

      <footer className="border-t">
        <div className="mx-auto max-w-6xl px-4 py-8 text-sm text-muted-foreground sm:px-6">
          © {new Date().getFullYear()} XHire — Explainable AI recruitment platform.
        </div>
      </footer>
    </div>
  );
}
