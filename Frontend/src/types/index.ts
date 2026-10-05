export type UserRole = "candidate" | "recruiter";

export interface User {
  id: string;
  email: string;
  fullName: string;
  role: UserRole;
  avatarUrl?: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface LoginPayload {
  email: string;
  password: string;
}

export interface RegisterPayload extends LoginPayload {
  fullName: string;
  role: UserRole;
}

export interface CandidateProfile {
  id: string;
  userId: string;
  fullName: string;
  email: string;
  phone?: string;
  headline?: string;
  location?: string;
  experienceYears?: number;
  skills: string[];
  education?: string;
  about?: string;
  linkedinUrl?: string;
  githubUrl?: string;
  resumeId?: string;
}

export interface RecruiterProfile {
  id: string;
  userId: string;
  fullName: string;
  email: string;
  companyName: string;
  companyWebsite?: string;
  companySize?: string;
  industry?: string;
  location?: string;
  about?: string;
}

export type JobType = "full-time" | "part-time" | "internship" | "contract";
export type WorkMode = "onsite" | "remote" | "hybrid";

export interface Job {
  id: string;
  title: string;
  companyName: string;
  location: string;
  workMode: WorkMode;
  jobType: JobType;
  salaryMin?: number;
  salaryMax?: number;
  experienceMin?: number;
  description: string;
  responsibilities?: string[];
  requiredSkills: string[];
  postedAt: string;
  status: "open" | "closed" | "draft";
  applicantCount?: number;
  recruiterId?: string;
}

export interface JobFilters {
  search?: string;
  location?: string;
  jobType?: JobType | "all";
  workMode?: WorkMode | "all";
  minExperience?: number;
}

export type ApplicationStatus =
  | "applied"
  | "under-review"
  | "shortlisted"
  | "interview"
  | "rejected"
  | "hired";

export interface Application {
  id: string;
  jobId: string;
  jobTitle: string;
  companyName: string;
  candidateId: string;
  candidateName?: string;
  status: ApplicationStatus;
  appliedAt: string;
  updatedAt?: string;
  matchScore?: number;
  timeline?: { status: ApplicationStatus; at: string; note?: string }[];
}

export interface Resume {
  id: string;
  candidateId: string;
  fileName: string;
  fileUrl?: string;
  uploadedAt: string;
  sizeKb?: number;
}

export interface ResumeAnalysis {
  resumeId: string;
  overallScore: number;
  sections: { name: string; score: number; feedback: string }[];
  extractedSkills: string[];
  strengths: string[];
  improvements: string[];
}

export interface SkillGap {
  skill: string;
  required: number;
  current: number;
  priority: "high" | "medium" | "low";
}

export interface MatchExplanation {
  jobId: string;
  candidateId: string;
  matchScore: number;
  summary: string;
  factors: { label: string; weight: number; contribution: number; detail: string }[];
  matchedSkills: string[];
  missingSkills: string[];
}

export interface RankedCandidate {
  candidateId: string;
  candidateName: string;
  headline?: string;
  matchScore: number;
  applicationId: string;
  topReasons: string[];
  matchedSkills: string[];
  missingSkills: string[];
}
