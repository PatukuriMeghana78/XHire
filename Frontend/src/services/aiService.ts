import { api, USE_MOCKS } from "@/lib/api/client";
import { mockDelay, mockMatchExplanation, mockRankedCandidates, mockSkillGaps } from "@/mocks/fixtures";
import type { MatchExplanation, RankedCandidate, SkillGap } from "@/types";

/**
 * All explainable-AI output is produced by the Go backend.
 * This layer only transports it — no inference happens in the browser.
 */
const routes = {
  skillGaps: "/api/v1/candidate/skill-gaps",
  jobMatch: (jobId: string) => `/api/v1/candidate/jobs/${jobId}/match`,
  ranking: (jobId: string) => `/api/v1/recruiter/jobs/${jobId}/ranking`,
  rankingExplanation: (jobId: string, candidateId: string) =>
    `/api/v1/recruiter/jobs/${jobId}/candidates/${candidateId}/explanation`,
};

export const aiService = {
  async getSkillGaps(jobId?: string): Promise<SkillGap[]> {
    if (USE_MOCKS) return mockDelay(mockSkillGaps, 600);
    return api.get<SkillGap[]>(routes.skillGaps, { query: { job_id: jobId } });
  },

  async getJobMatchExplanation(jobId: string): Promise<MatchExplanation> {
    if (USE_MOCKS) return mockDelay({ ...mockMatchExplanation, jobId }, 600);
    return api.get<MatchExplanation>(routes.jobMatch(jobId));
  },

  async getCandidateRanking(jobId: string): Promise<RankedCandidate[]> {
    if (USE_MOCKS) return mockDelay(mockRankedCandidates, 600);
    return api.get<RankedCandidate[]>(routes.ranking(jobId));
  },

  async getRankingExplanation(jobId: string, candidateId: string): Promise<MatchExplanation> {
    if (USE_MOCKS) return mockDelay({ ...mockMatchExplanation, jobId, candidateId }, 600);
    return api.get<MatchExplanation>(routes.rankingExplanation(jobId, candidateId));
  },
};
