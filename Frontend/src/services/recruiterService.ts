import { api, USE_MOCKS } from "@/lib/api/client";
import { mockDelay, mockRecruiterProfile } from "@/mocks/fixtures";
import type { RecruiterProfile } from "@/types";

const routes = {
  profile: "/api/v1/recruiter/profile",
};

export const recruiterService = {
  async getMyProfile(): Promise<RecruiterProfile> {
    if (USE_MOCKS) return mockDelay(mockRecruiterProfile);
    return api.get<RecruiterProfile>(routes.profile);
  },

  async updateMyProfile(payload: Partial<RecruiterProfile>): Promise<RecruiterProfile> {
    if (USE_MOCKS) return mockDelay({ ...mockRecruiterProfile, ...payload });
    return api.put<RecruiterProfile>(routes.profile, payload);
  },
};
