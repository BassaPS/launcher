import { AppService } from '@bindings/cyrene-launcher/internal/app-service';
import { PlatformService } from '@bindings/cyrene-launcher/internal/platform-service';
import { nextReleaseEvaluation, nextStatusEvaluation, releaseAge, statusAge, type Age } from './presentation';

export interface ResourceState<T> {
  phase: 'loading' | 'success' | 'stale' | 'unavailable';
  data: T | null;
  error: boolean;
  expired: boolean;
  now: number;
}

interface Cancellable<T> extends PromiseLike<T> { cancel(): unknown }

// A module-owned resource shares one call across mounted consumers. A consumer
// leaving never cancels another consumer's request. Cleanup is deferred one task
// to preserve React StrictMode's immediate setup/cleanup/setup cycle.
function createResource<T>(
  load: () => Cancellable<T>,
  age: (data: T, now: number) => Age,
  nextEvaluation: (data: T, now: number) => number | undefined,
  pollMs?: number,
) {
  const initial: ResourceState<T> = { phase: 'loading', data: null, error: false, expired: false, now: 0 };
  let state = initial;
  let data: T | null = null;
  let failed = false;
  let active = false;
  let sequence = 0;
  let lastActivity = -Infinity;
  let request: Cancellable<T> | undefined;
  let wake: ReturnType<typeof setTimeout> | undefined;
  let expiry: ReturnType<typeof setTimeout> | undefined;
  let cleanup: ReturnType<typeof setTimeout> | undefined;
  const listeners = new Set<() => void>();

  function emit() {
    const now = Date.now();
    const freshness = data === null ? undefined : age(data, now);
    const expired = freshness === 'expired';
    const phase = request ? 'loading' : failed || expired || data === null ? 'unavailable'
      : freshness === 'stale' ? 'stale' : 'success';
    state = { phase, data: expired ? null : data, error: failed, expired, now };
    for (const listener of listeners) listener();
  }

  function scheduleExpiry() {
    clearTimeout(expiry);
    if (data === null || !active || document.hidden) return;
    const next = nextEvaluation(data, Date.now());
    if (next !== undefined) expiry = setTimeout(() => { emit(); scheduleExpiry(); }, Math.max(0, next - Date.now()));
  }

  async function refresh() {
    if (!active || !listeners.size || document.hidden || request) return;
    clearTimeout(wake);
    const current = ++sequence;
    lastActivity = Date.now();
    failed = false;
    try {
      request = load();
      emit();
      const result = await request;
      if (!active || current !== sequence) return;
      data = result;
    } catch {
      if (!active || current !== sequence) return;
      failed = true;
    }
    if (!active || current !== sequence) return;
    request = undefined;
    lastActivity = Date.now();
    emit();
    scheduleExpiry();
    if (pollMs !== undefined) wake = setTimeout(() => void refresh(), pollMs);
  }

  function pause() {
    clearTimeout(wake);
    clearTimeout(expiry);
    sequence++;
    const pending = request;
    request = undefined;
    if (pending) void Promise.resolve(pending.cancel()).catch(() => {});
  }

  function resume() {
    pause();
    emit();
    if (!active || document.hidden) return;
    scheduleExpiry();
    if (pollMs !== undefined || data === null) {
      const cooldown = pollMs ?? 60_000;
      wake = setTimeout(() => void refresh(), Math.max(0, cooldown - (Date.now() - lastActivity)));
    }
  }

  return {
    getSnapshot: () => state,
    getServerSnapshot: () => initial,
    refresh: () => void refresh(),
    subscribe(listener: () => void) {
      clearTimeout(cleanup);
      listeners.add(listener);
      if (!active) {
        active = true;
        document.addEventListener('visibilitychange', resume);
        wake = setTimeout(resume, 0);
      }
      return () => {
        listeners.delete(listener);
        if (listeners.size) return;
        cleanup = setTimeout(() => {
          if (listeners.size) return;
          active = false;
          pause();
          document.removeEventListener('visibilitychange', resume);
        }, 0);
      };
    },
  };
}

export const statusResource = createResource(PlatformService.GetStatus, statusAge, nextStatusEvaluation, 60_000);
export const releaseResource = createResource(PlatformService.GetLatestRelease, releaseAge, nextReleaseEvaluation);
export const installedVersionResource = createResource(
  () => AppService.GetCurrentLauncherVersion().then(([ok, version]) => {
    if (!ok || !version.trim()) throw new Error('Installed version unavailable');
    return version;
  }),
  () => 'fresh',
  () => undefined,
);
