'use client';

import { RefreshCw, X } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import {
  fetchBuildVersion,
  getBuildVersionId,
  VERSION_POLL_INTERVAL_MS,
} from '@/lib/versionPolling';

const loadedBuildHash = process.env.NEXT_PUBLIC_BUILD_HASH || null;

export function VersionUpdateBanner() {
  const loadedVersionIdRef = useRef<string | null>(loadedBuildHash);
  const [availableVersionId, setAvailableVersionId] = useState<string | null>(null);
  const [dismissedVersionId, setDismissedVersionId] = useState<string | null>(null);

  useEffect(() => {
    let stopped = false;
    let timerId: number | undefined;
    let controller: AbortController | null = null;

    const scheduleNextPoll = () => {
      timerId = window.setTimeout(pollForVersion, VERSION_POLL_INTERVAL_MS);
    };

    const pollForVersion = async () => {
      controller = new AbortController();

      try {
        const version = await fetchBuildVersion(controller.signal);
        const nextVersionId = getBuildVersionId(version);

        if (stopped || !nextVersionId) {
          return;
        }

        if (!loadedVersionIdRef.current) {
          loadedVersionIdRef.current = nextVersionId;
          return;
        }

        if (nextVersionId !== loadedVersionIdRef.current) {
          setAvailableVersionId(nextVersionId);
        }
      } catch (error) {
        if (error instanceof Error && error.name === 'AbortError') {
          return;
        }
      } finally {
        if (!stopped) {
          scheduleNextPoll();
        }
      }
    };

    pollForVersion();

    return () => {
      stopped = true;
      controller?.abort();

      if (timerId) {
        window.clearTimeout(timerId);
      }
    };
  }, []);

  const isVisible = Boolean(availableVersionId && availableVersionId !== dismissedVersionId);

  if (!isVisible) {
    return null;
  }

  return (
    <div
      className="fixed bottom-4 left-4 right-4 z-50 mx-auto flex max-w-md items-center gap-3 rounded-md border border-slate-200 bg-white p-3 text-sm text-slate-700 shadow-lg"
      role="status"
      aria-live="polite"
    >
      <div className="min-w-0 flex-1">
        <p className="font-medium text-slate-900">A new version is available.</p>
      </div>
      <button
        type="button"
        onClick={() => window.location.reload()}
        className="inline-flex h-9 items-center gap-2 rounded-md bg-slate-900 px-3 text-sm font-medium text-white hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:ring-offset-2"
      >
        <RefreshCw className="h-4 w-4" aria-hidden="true" />
        Refresh
      </button>
      <button
        type="button"
        onClick={() => setDismissedVersionId(availableVersionId)}
        className="inline-flex h-9 w-9 items-center justify-center rounded-md text-slate-500 hover:bg-slate-100 hover:text-slate-700 focus:outline-none focus:ring-2 focus:ring-slate-500 focus:ring-offset-2"
        aria-label="Dismiss update notification"
      >
        <X className="h-4 w-4" aria-hidden="true" />
      </button>
    </div>
  );
}
