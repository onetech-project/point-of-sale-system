export interface BuildVersion {
  buildHash?: string;
  buildTime?: string;
  appVersion?: string;
  commit?: string | null;
}

export const VERSION_ENDPOINT = '/version.json';
export const VERSION_POLL_INTERVAL_MS = 5 * 60 * 1000;

export const getBuildVersionId = (version: BuildVersion | null): string | null => {
  if (!version) {
    return null;
  }

  return version.buildHash || version.commit || version.buildTime || null;
};

export const fetchBuildVersion = async (signal?: AbortSignal): Promise<BuildVersion | null> => {
  const response = await fetch(`${VERSION_ENDPOINT}?t=${Date.now()}`, {
    cache: 'no-store',
    headers: {
      'Cache-Control': 'no-cache',
    },
    signal,
  });

  if (!response.ok) {
    return null;
  }

  return response.json();
};
