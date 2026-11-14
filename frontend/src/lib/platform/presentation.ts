import type { ReleaseSnapshot, StatusService, StatusSnapshot } from '@bindings/cyrene-launcher/internal/platform-service/models';

export const statusFreshMs = 450_000;
export const clockSkewMs = 60_000;
export const releaseFreshMs = 600_000;
export const releaseRetentionMs = 7 * 86_400_000;
export const maxTimerDelayMs = 2_147_483_647;
export const primaryServices = [
  { id: 'portal', monitoringType: 'http' },
  { id: 'account', monitoringType: 'http' },
  { id: 'download', monitoringType: 'http' },
  { id: 'docs', monitoringType: 'http' },
  { id: 'm7h-http', monitoringType: 'http' },
  { id: 'm7h-game', monitoringType: 'udp' },
] as const;

export type Age = 'fresh' | 'stale' | 'expired';

function observed(value: string | null, now: number): boolean {
  if (!value) return false;
  const age = now - Date.parse(value);
  return Number.isFinite(age) && age <= statusFreshMs && age >= -clockSkewMs;
}

export function serviceView(data: StatusSnapshot, service: StatusService | undefined, now: number) {
  const staleObservation = !observed(service?.lastCheckedAt ?? null, now);
  const base = { staleObservation, insufficientData: (service?.samples24H ?? 0) < 20, declared: false };
  if (!service) return { ...base, status: 'unknown' };
  if (!service.enabled || service.status === 'not_monitored') return { ...base, status: 'not_monitored' };
  const expected = primaryServices.find(item => item.id === service.id);
  if (expected && expected.monitoringType !== service.monitoringType) return { ...base, status: 'unknown' };
  // The provider applies active maintenance and unresolved incident overrides
  // after checking observations. Retain that declaration while its API snapshot
  // is current, and keep the old observation explicitly marked as old.
  const declared = service.status === 'maintenance'
    || service.hasIncident && ['degraded', 'partial_outage', 'major_outage'].includes(service.status);
  if (declared && observed(data.generatedAt, now)) return { ...base, status: service.status, declared: true };
  return { ...base, status: staleObservation ? 'unknown' : service.status };
}

export function snapshotStatus(data: StatusSnapshot, now: number): string {
  if (!observed(data.generatedAt, now)) return 'unknown';
  if (data.overallStatus !== 'operational') return data.overallStatus;
  if (!observed(data.updatedAt, now) || !data.services.some(service => service.enabled)
    || primaryServices.some(({ id }) => !data.services.some(service => service.id === id))
    || data.services.some(service => service.enabled && serviceView(data, service, now).status !== 'operational')) return 'unknown';
  return 'operational';
}

export function statusAge(data: StatusSnapshot, now: number): Age {
  return !observed(data.generatedAt, now)
    || data.services.some(service => service.enabled && serviceView(data, service, now).staleObservation) ? 'stale' : 'fresh';
}

export function releaseAge(data: ReleaseSnapshot, now: number): Age {
  const age = now - Date.parse(data.fetchedAt);
  if (!Number.isFinite(age) || age >= releaseRetentionMs || age < -clockSkewMs) return 'expired';
  return data.stale || age >= releaseFreshMs ? 'stale' : 'fresh';
}

function nextTime(times: number[], now: number): number | undefined {
  const future = times.filter(time => Number.isFinite(time) && time > now);
  return future.length ? Math.min(...future, now + maxTimerDelayMs) : undefined;
}

export function nextStatusEvaluation(data: StatusSnapshot, now: number): number | undefined {
  const times = [data.generatedAt, data.updatedAt, ...data.services.filter(service => service.enabled).map(service => service.lastCheckedAt)]
    .filter((value): value is string => value !== null).map(Date.parse);
  return nextTime(times.flatMap(time => [time - clockSkewMs, time + statusFreshMs + 1]), now);
}

export function nextReleaseEvaluation(data: ReleaseSnapshot, now: number): number | undefined {
  const time = Date.parse(data.fetchedAt);
  return nextTime([time + releaseFreshMs, time + releaseRetentionMs], now);
}
