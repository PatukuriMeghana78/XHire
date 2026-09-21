import { api, USE_MOCKS } from "@/lib/api/client";
import { mockCandidateDirectory, mockCandidateProfile, mockDelay } from "@/mocks/fixtures";
import type { CandidateProfile } from "@/types";

const routes = {
  profile: "/api/v1/candidate/profile",
  candidate: (id: string) => `/api/v1/candidates/${id}`,
};

export const candidateService = {
  async getMyProfile(): Promise<CandidateProfile> {
    if (USE_MOCKS) return mockDelay(mockCandidateProfile);
    return api.get<CandidateProfile>(routes.profile);
  },

  async updateMyProfile(payload: Partial<CandidateProfile>): Promise<CandidateProfile> {
    if (USE_MOCKS) return mockDelay({ ...mockCandidateProfile, ...payload });
    return api.put<CandidateProfile>(routes.profile, payload);
  },

  async getCandidate(id: string): Promise<CandidateProfile> {
    if (USE_MOCKS)
      return mockDelay(
        mockCandidateDirectory.find((c) => c.id === id) ?? mockCandidateDirectory[0]!,
      );
    return api.get<CandidateProfile>(routes.candidate(id));
  },
};
