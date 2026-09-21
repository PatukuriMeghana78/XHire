import { api, USE_MOCKS } from "@/lib/api/client";
import { mockApplications, mockDelay } from "@/mocks/fixtures";
import type { Application, ApplicationStatus } from "@/types";

const routes = {
  myApplications: "/api/v1/candidate/applications",
  application: (id: string) => `/api/v1/applications/${id}`,
  apply: (jobId: string) => `/api/v1/jobs/${jobId}/apply`,
  jobApplications: (jobId: string) => `/api/v1/recruiter/jobs/${jobId}/applications`,
  recruiterApplications: "/api/v1/recruiter/applications",
  status: (id: string) => `/api/v1/recruiter/applications/${id}/status`,
};

export const applicationService = {
  async listMyApplications(): Promise<Application[]> {
    if (USE_MOCKS) return mockDelay(mockApplications.filter((a) => a.candidateId === "c-1"));
    return api.get<Application[]>(routes.myApplications);
  },

  async getApplication(id: string): Promise<Application> {
    if (USE_MOCKS)
      return mockDelay(mockApplications.find((a) => a.id === id) ?? mockApplications[0]!);
    return api.get<Application>(routes.application(id));
  },

  async applyToJob(jobId: string, payload: { coverLetter?: string; resumeId?: string } = {}) {
    if (USE_MOCKS)
      return mockDelay({
        ...mockApplications[0]!,
        id: `a-${Date.now()}`,
        jobId,
        status: "applied" as ApplicationStatus,
        appliedAt: new Date().toISOString(),
      });
    return api.post<Application>(routes.apply(jobId), payload);
  },

  async listApplicationsForJob(jobId: string): Promise<Application[]> {
    if (USE_MOCKS) return mockDelay(mockApplications.filter((a) => a.jobId === jobId));
    return api.get<Application[]>(routes.jobApplications(jobId));
  },

  async listRecruiterApplications(): Promise<Application[]> {
    if (USE_MOCKS) return mockDelay(mockApplications);
    return api.get<Application[]>(routes.recruiterApplications);
  },

  async updateStatus(id: string, status: ApplicationStatus): Promise<Application> {
    if (USE_MOCKS)
      return mockDelay({
        ...(mockApplications.find((a) => a.id === id) ?? mockApplications[0]!),
        status,
      });
    return api.patch<Application>(routes.status(id), { status });
  },
};
