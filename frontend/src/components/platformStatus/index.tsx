import { useTranslation } from 'react-i18next';
import { usePlatformStatus } from '@/hooks/usePlatformInfo';
import { PlatformLink } from '@/components/platformLinks';
import { statusAge } from '@/lib/platform/presentation';

const serviceIDs = ['m7h-http', 'm7h-game', 'account', 'docs'] as const;

export default function PlatformStatus() {
  const { t, i18n } = useTranslation();
  const { status, services, overallStatus, refreshStatus } = usePlatformStatus();
  return <section className="bassaps-panel bassaps-platform-status" aria-label={t('platform.statusTitle')} aria-busy={status.phase === 'loading'}>
    <div className="bassaps-panel-heading"><h2>{t('platform.statusTitle')}</h2>
      <button type="button" className="bassaps-platform-link" disabled={status.phase === 'loading'} onClick={refreshStatus}>{t('platform.refresh')}</button></div>
    <div role="status" aria-live="polite">
      {status.phase === 'loading' && <p>{t('platform.loading')}</p>}
      {status.error && <p>{t('platform.statusError')}</p>}
      {status.data && <>
        {statusAge(status.data, status.now) === 'stale' && <p>{t('platform.oldStatus')}</p>}
        <p className="bassaps-platform-overall"><span>{t('platform.overall')}</span><strong data-state={overallStatus}>{t(`platform.states.${overallStatus}`)}</strong></p>
        <ul className="bassaps-platform-rows">{serviceIDs.map(id => {
          const view = services.find(service => service.id === id)!;
          const observed = status.data?.services.find(service => service.id === id)?.lastCheckedAt;
          return <li key={id} data-service={id}>
            <div><span>{t(`platform.targets.${id}`)}</span><strong data-state={view.status}>{t(`platform.states.${view.status}`)}</strong></div>
            {view.status !== 'not_monitored' && <small>
              {observed ? <>{t('platform.observed')}: <time dateTime={observed} title={observed}>{new Date(observed).toLocaleString(i18n.resolvedLanguage)}</time></> : t('platform.noObservation')}
              {view.staleObservation && observed && <> · {t('platform.oldObservation')}</>}
              {view.insufficientData && <> · {t('platform.insufficient')}</>}
              {view.declared && <> · {t('platform.declared')}</>}
            </small>}
          </li>;
        })}</ul>
      </>}
    </div>
    <p className="bassaps-platform-note">{t('platform.httpOnly')}</p>
    <PlatformLink id="status" />
  </section>;
}
