import { test } from 'node:test';
import assert from 'node:assert/strict';
import {
  primaryServices, serviceView, snapshotStatus, releaseAge, nextStatusEvaluation,
  nextReleaseEvaluation, statusFreshMs, releaseFreshMs, releaseRetentionMs, maxTimerDelayMs,
} from '../src/lib/platform/presentation.ts';

const now = Date.parse('2026-11-13T12:00:00.000Z');
const time = new Date(now).toISOString();
function snapshot() {
  return {
    overallStatus: 'operational', updatedAt: time, generatedAt: time,
    services: primaryServices.map(({ id, monitoringType }) => ({
      id, name: id, monitoringType, enabled: monitoringType === 'http',
      status: monitoringType === 'http' ? 'operational' : 'not_monitored',
      lastCheckedAt: monitoringType === 'http' ? time : null,
      responseTimeMs: null, samples24H: 1, hasIncident: false,
    })),
  };
}

test('status observation boundary and insufficient samples', () => {
  const data = snapshot();
  assert.equal(serviceView(data, data.services[0], now + statusFreshMs).status, 'operational');
  assert.equal(serviceView(data, data.services[0], now + statusFreshMs + 1).status, 'unknown');
  assert.equal(serviceView(data, data.services[0], now).insufficientData, true);
  assert.equal(nextStatusEvaluation(data, now), now + statusFreshMs + 1);
  assert.equal(snapshotStatus(data, now + statusFreshMs + 1), 'unknown');
});

test('disabled docs, UDP, missing targets, future time and probe mismatch', () => {
  const data = snapshot();
  const docs = data.services.find(s => s.id === 'docs')!;
  docs.enabled = false;
  assert.equal(serviceView(data, docs, now).status, 'not_monitored');
  assert.equal(serviceView(data, data.services.at(-1), now).status, 'not_monitored');
  assert.equal(serviceView(data, undefined, now).status, 'unknown');
  docs.enabled = true;
  docs.lastCheckedAt = new Date(now + 60_001).toISOString();
  assert.equal(serviceView(data, docs, now).status, 'unknown');
  docs.lastCheckedAt = new Date(now + 60_000).toISOString();
  assert.equal(serviceView(data, docs, now).status, 'operational');
  docs.monitoringType = 'udp';
  assert.equal(serviceView(data, docs, now).status, 'unknown');
  data.services = data.services.filter(s => s.id !== 'account');
  assert.equal(snapshotStatus(data, now), 'unknown');
});

test('fresh declarations preserve maintenance and known incidents with old observations', () => {
  const data = snapshot();
  const service = data.services[0];
  service.lastCheckedAt = new Date(now - statusFreshMs - 1).toISOString();
  service.status = 'maintenance';
  const maintenance = serviceView(data, service, now);
  assert.equal(maintenance.status, 'maintenance');
  assert.equal(maintenance.staleObservation, true);
  assert.equal(maintenance.declared, true);
  service.status = 'major_outage';
  assert.equal(serviceView(data, service, now).status, 'unknown');
  service.hasIncident = true;
  assert.equal(serviceView(data, service, now).status, 'major_outage');
  assert.equal(serviceView(data, service, now + statusFreshMs + 1).status, 'unknown');
});

test('release freshness and retention boundaries preserve provider timestamp', () => {
  const release = { latest: null, recommendedAsset: null, assetState: 'not_provided', fetchedAt: time, stale: false, downloadUrl: 'https://download.bassaps.com/' };
  assert.equal(releaseAge(release, now + releaseFreshMs - 1), 'fresh');
  assert.equal(releaseAge(release, now + releaseFreshMs), 'stale');
  assert.equal(releaseAge(release, now + releaseRetentionMs - 1), 'stale');
  assert.equal(releaseAge(release, now + releaseRetentionMs), 'expired');
  assert.equal(releaseAge(release, now - 60_000), 'fresh');
  assert.equal(releaseAge(release, now - 60_001), 'expired');
  assert.equal(nextReleaseEvaluation(release, now), now + releaseFreshMs);
  assert.equal(nextReleaseEvaluation(release, now + releaseFreshMs), now + releaseRetentionMs);
  release.stale = true;
  assert.equal(releaseAge(release, now), 'stale');
  assert.equal(release.fetchedAt, time);
});

test('distant timestamps stay within the browser timer limit', () => {
  const data = snapshot();
  const future = '2099-01-01T00:00:00.000Z';
  data.generatedAt = future;
  data.updatedAt = future;
  for (const service of data.services) service.lastCheckedAt = future;
  assert.equal(snapshotStatus(data, now), 'unknown');
  assert.equal(nextStatusEvaluation(data, now), now + maxTimerDelayMs);
  const release = { latest: null, recommendedAsset: null, assetState: 'none', fetchedAt: future, stale: false, downloadUrl: 'https://download.bassaps.com/' };
  assert.equal(releaseAge(release, now), 'expired');
  assert.equal(nextReleaseEvaluation(release, now), now + maxTimerDelayMs);
});
