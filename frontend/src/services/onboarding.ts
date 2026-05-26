import apiClient from './api';

export interface OnboardingProgress {
  completed: boolean;
  completed_at?: string;
}

export interface CompleteOnboardingResponse {
  success: true;
  completed: true;
  completed_at: string;
}

export const onboardingService = {
  getProgress(tourKey: string, tourVersion: number): Promise<OnboardingProgress> {
    const params = new URLSearchParams({
      tour_key: tourKey,
      tour_version: String(tourVersion),
    });
    return apiClient.get<OnboardingProgress>(`/api/v1/users/onboarding/progress?${params}`);
  },

  complete(tourKey: string, tourVersion: number): Promise<CompleteOnboardingResponse> {
    return apiClient.post<CompleteOnboardingResponse>('/api/v1/users/onboarding/complete', {
      tour_key: tourKey,
      tour_version: tourVersion,
    });
  },
};
