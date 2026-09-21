import { api, USE_MOCKS } from "@/lib/api/client";
import { mockDelay, mockJobs } from "@/mocks/fixtures";
import type { Job, JobFilters } from "@/types";

const routes = {
  jobs: "/api/v1/jobs",
  job: (id: string) => `/api/v1/jobs/${id}`,
  recruiterJobs: "/api/v1/recruiter/jobs",
};

function applyFilters(jobs: Job[], filters: JobFilters = {}): Job[] {
  const term = filters.search?.toLowerCase().trim();
  return jobs.filter((job) => {
    if (
      term &&
      !`${job.title} ${job.companyName} ${job.requiredSkills.join(" ")}`
        .toLowerCase()
        .includes(term)
    )
      return false;
    if (filters.location && !job.location.toLowerCase().includes(filters.location.toLowerCase()))
      return false;
    if (filters.jobType && filters.jobType !== "all" && job.jobType !== filters.jobType)
      return false;
    if (filters.workMode && filters.workMode !== "all" && job.workMode !== filters.workMode)
      return false;
    if (filters.minExperience !== undefined && (job.experienceMin ?? 0) > filters.minExperience)
      return false;
    return true;
  });
}

export const jobService = {
  async listJobs(filters: JobFilters = {}): Promise<Job[]> {
    if (USE_MOCKS) {
      return mockDelay(applyFilters(mockJobs.filter((j) => j.status === "open"), filters));
    }
    return api.get<Job[]>(routes.jobs, {
      query: {
        search: filters.search,
        location: filters.location,
        job_type: filters.jobType === "all" ? undefined : filters.jobType,
        work_mode: filters.workMode === "all" ? undefined : filters.workMode,
        min_experience: filters.minExperience,
      },
    });
  },

  async getJob(id: string): Promise<Job> {
    if (USE_MOCKS) {
      const job = mockJobs.find((j) => j.id === id) ?? mockJobs[0]!;
      return mockDelay(job);
    }
    return api.get<Job>(routes.job(id));
  },

  async listRecruiterJobs(): Promise<Job[]> {
    if (USE_MOCKS) return mockDelay(mockJobs.filter((j) => j.recruiterId === "rec-1"));
    return api.get<Job[]>(routes.recruiterJobs);
  },

  async createJob(payload: Partial<Job>): Promise<Job> {
    if (USE_MOCKS)
      return mockDelay({
        ...mockJobs[0]!,
        ...payload,
        id: `j-${Date.now()}`,
        postedAt: new Date().toISOString(),
      } as Job);
    return api.post<Job>(routes.jobs, payload);
  },

  async updateJob(id: string, payload: Partial<Job>): Promise<Job> {
    if (USE_MOCKS) return mockDelay({ ...(mockJobs.find((j) => j.id === id) ?? mockJobs[0]!), ...payload } as Job);
    return api.put<Job>(routes.job(id), payload);
  },

  async deleteJob(id: string): Promise<void> {
    if (USE_MOCKS) return mockDelay(undefined);
    return api.delete<void>(routes.job(id));
  },
};
