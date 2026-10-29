import { api, apiDownload, USE_MOCKS } from "@/lib/api/client";
import { mockDelay, mockResume, mockResumeAnalysis } from "@/mocks/fixtures";
import type { Resume, ResumeAnalysis } from "@/types";

const routes = {
  myResume: "/api/v1/candidate/resume",
  upload: "/api/v1/candidate/resume",
  analysis: (resumeId: string) => `/api/v1/resumes/${resumeId}/analysis`,
  candidateResume: (candidateId: string) => `/api/v1/candidates/${candidateId}/resume`,
  download: (resumeId: string) => `/api/v1/resumes/${resumeId}/download`,
};

export const resumeService = {
  async getMyResume(): Promise<Resume | null> {
    if (USE_MOCKS) return mockDelay(mockResume);
    return api.get<Resume | null>(routes.myResume);
  },

  async uploadResume(file: File): Promise<Resume> {
    if (USE_MOCKS)
      return mockDelay(
        {
          ...mockResume,
          fileName: file.name,
          sizeKb: Math.round(file.size / 1024),
          uploadedAt: new Date().toISOString(),
        },
        900,
      );
    const form = new FormData();
    form.append("resume", file);
    return api.post<Resume>(routes.upload, form);
  },

  async getAnalysis(resumeId: string): Promise<ResumeAnalysis> {
    if (USE_MOCKS) return mockDelay(mockResumeAnalysis, 700);
    return api.get<ResumeAnalysis>(routes.analysis(resumeId));
  },

  async getCandidateResume(candidateId: string): Promise<Resume> {
    if (USE_MOCKS) return mockDelay({ ...mockResume, candidateId });
    return api.get<Resume>(routes.candidateResume(candidateId));
  },

  async downloadResume(resumeId: string, fileName?: string): Promise<void> {
    if (USE_MOCKS) return;
    return apiDownload(routes.download(resumeId), fileName);
  },
};
