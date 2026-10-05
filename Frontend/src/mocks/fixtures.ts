/**
 * TEMPORARY UI FIXTURES ONLY.
 * Used when VITE_API_BASE_URL is not configured. Delete this folder once the
 * Go backend endpoints are live — nothing here is business logic.
 */
import type {
  Application,
  CandidateProfile,
  Job,
  MatchExplanation,
  RankedCandidate,
  RecruiterProfile,
  Resume,
  ResumeAnalysis,
  SkillGap,
  User,
} from "@/types";

export const mockDelay = <T>(value: T, ms = 420): Promise<T> =>
  new Promise((resolve) => setTimeout(() => resolve(value), ms));

export const mockCandidateUser: User = {
  id: "u-cand-1",
  email: "aarav.sharma@example.com",
  fullName: "Aarav Sharma",
  role: "candidate",
};

export const mockRecruiterUser: User = {
  id: "u-rec-1",
  email: "priya.nair@northwind.io",
  fullName: "Priya Nair",
  role: "recruiter",
};

export const mockCandidateProfile: CandidateProfile = {
  id: "c-1",
  userId: "u-cand-1",
  fullName: "Aarav Sharma",
  email: "aarav.sharma@example.com",
  phone: "+91 98765 43210",
  headline: "Final-year CSE student · Backend & ML enthusiast",
  location: "Hyderabad, India",
  experienceYears: 1,
  skills: ["Go", "React", "TypeScript", "PostgreSQL", "Docker", "Python"],
  education: "B.Tech Computer Science, 2026",
  about:
    "Backend-leaning engineer with internship experience building REST services in Go and dashboards in React.",
  linkedinUrl: "https://linkedin.com/in/example",
  githubUrl: "https://github.com/example",
  resumeId: "r-1",
};

export const mockRecruiterProfile: RecruiterProfile = {
  id: "rec-1",
  userId: "u-rec-1",
  fullName: "Priya Nair",
  email: "priya.nair@northwind.io",
  companyName: "Northwind Labs",
  companyWebsite: "https://northwind.io",
  companySize: "201-500",
  industry: "Enterprise Software",
  location: "Bengaluru, India",
  about: "We build data infrastructure for regulated industries.",
};

export const mockJobs: Job[] = [
  {
    id: "j-1",
    title: "Backend Engineer (Go)",
    companyName: "Northwind Labs",
    location: "Bengaluru, India",
    workMode: "hybrid",
    jobType: "full-time",
    salaryMin: 1200000,
    salaryMax: 1800000,
    experienceMin: 1,
    description:
      "Build and scale Go microservices powering our data platform. You will own services end to end, from API design to production observability.",
    responsibilities: [
      "Design and ship REST APIs in Go + Gin",
      "Model and optimise PostgreSQL schemas",
      "Own service reliability and on-call rotations",
    ],
    requiredSkills: ["Go", "PostgreSQL", "Docker", "REST APIs", "Kubernetes"],
    postedAt: "2026-09-12T09:00:00Z",
    status: "open",
    applicantCount: 42,
    recruiterId: "rec-1",
  },
  {
    id: "j-2",
    title: "Frontend Engineer, React",
    companyName: "Northwind Labs",
    location: "Remote, India",
    workMode: "remote",
    jobType: "full-time",
    salaryMin: 1000000,
    salaryMax: 1600000,
    experienceMin: 2,
    description:
      "Craft the analytics experience used by thousands of operators daily. Strong product sense and attention to interaction detail required.",
    responsibilities: [
      "Build accessible, performant React interfaces",
      "Partner with design on a shared component system",
    ],
    requiredSkills: ["React", "TypeScript", "Tailwind CSS", "TanStack Query", "Testing"],
    postedAt: "2026-09-15T09:00:00Z",
    status: "open",
    applicantCount: 27,
    recruiterId: "rec-1",
  },
  {
    id: "j-3",
    title: "Machine Learning Intern",
    companyName: "Vertex AI Systems",
    location: "Hyderabad, India",
    workMode: "onsite",
    jobType: "internship",
    salaryMin: 40000,
    salaryMax: 60000,
    experienceMin: 0,
    description:
      "Work with the applied research team on resume parsing and candidate ranking models, including explainability tooling.",
    requiredSkills: ["Python", "scikit-learn", "NLP", "Pandas", "SHAP"],
    postedAt: "2026-09-18T09:00:00Z",
    status: "open",
    applicantCount: 88,
    recruiterId: "rec-2",
  },
  {
    id: "j-4",
    title: "Platform Engineer",
    companyName: "Koto Cloud",
    location: "Pune, India",
    workMode: "hybrid",
    jobType: "contract",
    salaryMin: 900000,
    salaryMax: 1400000,
    experienceMin: 3,
    description:
      "Own CI/CD, infrastructure as code and developer tooling across a multi-region deployment footprint.",
    requiredSkills: ["Terraform", "AWS", "Go", "CI/CD", "Observability"],
    postedAt: "2026-09-08T09:00:00Z",
    status: "open",
    applicantCount: 15,
    recruiterId: "rec-3",
  },
  {
    id: "j-5",
    title: "Data Engineer",
    companyName: "Northwind Labs",
    location: "Bengaluru, India",
    workMode: "onsite",
    jobType: "full-time",
    salaryMin: 1400000,
    salaryMax: 2000000,
    experienceMin: 2,
    description:
      "Design batch and streaming pipelines that feed the XHire matching models with clean, governed data.",
    requiredSkills: ["Python", "Airflow", "PostgreSQL", "Kafka", "dbt"],
    postedAt: "2026-09-02T09:00:00Z",
    status: "closed",
    applicantCount: 61,
    recruiterId: "rec-1",
  },
];

export const mockApplications: Application[] = [
  {
    id: "a-1",
    jobId: "j-1",
    jobTitle: "Backend Engineer (Go)",
    companyName: "Northwind Labs",
    candidateId: "c-1",
    candidateName: "Aarav Sharma",
    status: "shortlisted",
    appliedAt: "2026-09-13T10:20:00Z",
    updatedAt: "2026-09-17T08:00:00Z",
    matchScore: 87,
    timeline: [
      { status: "applied", at: "2026-09-13T10:20:00Z" },
      { status: "under-review", at: "2026-09-15T12:00:00Z", note: "Resume screened by recruiter" },
      { status: "shortlisted", at: "2026-09-17T08:00:00Z", note: "Moved to technical round" },
    ],
  },
  {
    id: "a-2",
    jobId: "j-2",
    jobTitle: "Frontend Engineer, React",
    companyName: "Northwind Labs",
    candidateId: "c-1",
    candidateName: "Aarav Sharma",
    status: "under-review",
    appliedAt: "2026-09-16T09:00:00Z",
    matchScore: 74,
    timeline: [
      { status: "applied", at: "2026-09-16T09:00:00Z" },
      { status: "under-review", at: "2026-09-18T09:00:00Z" },
    ],
  },
  {
    id: "a-3",
    jobId: "j-3",
    jobTitle: "Machine Learning Intern",
    companyName: "Vertex AI Systems",
    candidateId: "c-1",
    candidateName: "Aarav Sharma",
    status: "rejected",
    appliedAt: "2026-08-30T09:00:00Z",
    matchScore: 52,
    timeline: [
      { status: "applied", at: "2026-08-30T09:00:00Z" },
      { status: "rejected", at: "2026-09-05T09:00:00Z", note: "Looking for stronger NLP depth" },
    ],
  },
  {
    id: "a-4",
    jobId: "j-1",
    jobTitle: "Backend Engineer (Go)",
    companyName: "Northwind Labs",
    candidateId: "c-2",
    candidateName: "Meera Iyer",
    status: "interview",
    appliedAt: "2026-09-14T09:00:00Z",
    matchScore: 91,
  },
  {
    id: "a-5",
    jobId: "j-1",
    jobTitle: "Backend Engineer (Go)",
    companyName: "Northwind Labs",
    candidateId: "c-3",
    candidateName: "Rohit Verma",
    status: "applied",
    appliedAt: "2026-09-16T09:00:00Z",
    matchScore: 68,
  },
];

export const mockResume: Resume = {
  id: "r-1",
  candidateId: "c-1",
  fileName: "aarav-sharma-resume.pdf",
  uploadedAt: "2026-09-10T11:00:00Z",
  sizeKb: 284,
};

export const mockResumeAnalysis: ResumeAnalysis = {
  resumeId: "r-1",
  overallScore: 78,
  sections: [
    { name: "Experience", score: 82, feedback: "Impact is described, but add measurable outcomes." },
    { name: "Skills", score: 88, feedback: "Strong backend coverage aligned with target roles." },
    { name: "Projects", score: 71, feedback: "Add links and describe your individual contribution." },
    { name: "Formatting", score: 65, feedback: "Two-column layout may confuse automated parsers." },
  ],
  extractedSkills: ["Go", "React", "TypeScript", "PostgreSQL", "Docker", "Python", "REST APIs"],
  strengths: [
    "Clear backend specialisation with production-style projects",
    "Consistent use of action verbs and concise bullets",
  ],
  improvements: [
    "Quantify results (latency, users, cost) in at least three bullets",
    "Move to a single-column, parser-friendly layout",
    "Add Kubernetes exposure to close the most common gap in your target roles",
  ],
};

export const mockSkillGaps: SkillGap[] = [
  { skill: "Kubernetes", required: 80, current: 25, priority: "high" },
  { skill: "System Design", required: 75, current: 45, priority: "high" },
  { skill: "Go", required: 85, current: 78, priority: "low" },
  { skill: "PostgreSQL", required: 70, current: 66, priority: "low" },
  { skill: "Testing", required: 70, current: 40, priority: "medium" },
  { skill: "Observability", required: 60, current: 30, priority: "medium" },
];

export const mockMatchExplanation: MatchExplanation = {
  jobId: "j-1",
  candidateId: "c-1",
  matchScore: 87,
  summary:
    "Strong overall fit. Backend language and database requirements are fully covered; container orchestration is the main gap holding the score below 90.",
  factors: [
    {
      label: "Skill overlap",
      weight: 0.4,
      contribution: 36,
      detail: "4 of 5 required skills found in your resume and profile.",
    },
    {
      label: "Experience level",
      weight: 0.25,
      contribution: 21,
      detail: "1 year against a 1+ year requirement — meets the minimum bar.",
    },
    {
      label: "Project relevance",
      weight: 0.2,
      contribution: 18,
      detail: "Two projects use Go services with PostgreSQL persistence.",
    },
    {
      label: "Location & work mode",
      weight: 0.15,
      contribution: 12,
      detail: "Hybrid in Bengaluru; you are open to relocation from Hyderabad.",
    },
  ],
  matchedSkills: ["Go", "PostgreSQL", "Docker", "REST APIs"],
  missingSkills: ["Kubernetes"],
};

export const mockRankedCandidates: RankedCandidate[] = [
  {
    candidateId: "c-2",
    candidateName: "Meera Iyer",
    headline: "Backend Engineer · 3 yrs · Go, Kubernetes",
    matchScore: 91,
    applicationId: "a-4",
    topReasons: [
      "All five required skills present, including Kubernetes",
      "3 years of production Go experience against a 1+ year bar",
      "Based in Bengaluru — matches hybrid requirement",
    ],
    matchedSkills: ["Go", "PostgreSQL", "Docker", "REST APIs", "Kubernetes"],
    missingSkills: [],
  },
  {
    candidateId: "c-1",
    candidateName: "Aarav Sharma",
    headline: "Final-year CSE · Backend & ML",
    matchScore: 87,
    applicationId: "a-1",
    topReasons: [
      "Four of five required skills matched",
      "Strong Go + PostgreSQL project portfolio",
      "Open to relocation for hybrid roles",
    ],
    matchedSkills: ["Go", "PostgreSQL", "Docker", "REST APIs"],
    missingSkills: ["Kubernetes"],
  },
  {
    candidateId: "c-3",
    candidateName: "Rohit Verma",
    headline: "Full-stack Developer · 2 yrs",
    matchScore: 68,
    applicationId: "a-5",
    topReasons: [
      "Go experience limited to side projects",
      "Strong REST API and Docker familiarity",
      "No exposure to container orchestration",
    ],
    matchedSkills: ["REST APIs", "Docker", "PostgreSQL"],
    missingSkills: ["Go", "Kubernetes"],
  },
];

export const mockCandidateDirectory: CandidateProfile[] = [
  mockCandidateProfile,
  {
    id: "c-2",
    userId: "u-cand-2",
    fullName: "Meera Iyer",
    email: "meera.iyer@example.com",
    headline: "Backend Engineer · 3 yrs · Go, Kubernetes",
    location: "Bengaluru, India",
    experienceYears: 3,
    skills: ["Go", "Kubernetes", "PostgreSQL", "Docker", "REST APIs", "gRPC"],
    education: "B.E. Information Science, 2023",
    about: "Platform-minded backend engineer who enjoys reliability work.",
    resumeId: "r-2",
  },
  {
    id: "c-3",
    userId: "u-cand-3",
    fullName: "Rohit Verma",
    email: "rohit.verma@example.com",
    headline: "Full-stack Developer · 2 yrs",
    location: "Pune, India",
    experienceYears: 2,
    skills: ["React", "Node.js", "PostgreSQL", "Docker", "REST APIs"],
    education: "B.Tech IT, 2024",
    about: "Product-focused developer shipping customer-facing features.",
    resumeId: "r-3",
  },
];
