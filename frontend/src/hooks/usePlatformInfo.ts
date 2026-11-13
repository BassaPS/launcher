import { useSyncExternalStore } from 'react';
import { installedVersionResource, releaseResource, statusResource } from '@/lib/platform/resources';
import { primaryServices, serviceView, snapshotStatus } from '@/lib/platform/presentation';

export function usePlatformInfo() {
  const status = useSyncExternalStore(statusResource.subscribe, statusResource.getSnapshot, statusResource.getServerSnapshot);
  const release = useSyncExternalStore(releaseResource.subscribe, releaseResource.getSnapshot, releaseResource.getServerSnapshot);
  const installedVersion = useSyncExternalStore(installedVersionResource.subscribe, installedVersionResource.getSnapshot, installedVersionResource.getServerSnapshot);
  return {
    status,
    release,
    installedVersion,
    overallStatus: status.data ? snapshotStatus(status.data, status.now) : 'unknown',
    services: primaryServices.map(({ id }) => ({
      id,
      ...serviceView(status.data ?? { generatedAt: '', overallStatus: 'unknown', updatedAt: null, services: [] },
        status.data?.services.find(service => service.id === id), status.now),
    })),
    refreshStatus: statusResource.refresh,
    refreshRelease: releaseResource.refresh,
  };
}
