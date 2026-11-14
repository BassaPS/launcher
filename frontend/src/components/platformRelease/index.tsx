import { useTranslation } from 'react-i18next';
import { usePlatformRelease } from '@/hooks/usePlatformInfo';
import { PlatformLink } from '@/components/platformLinks';
import { releaseAge } from '@/lib/platform/presentation';

export default function PlatformRelease() {
  const { t, i18n } = useTranslation();
  const { release, installedVersion, refreshRelease } = usePlatformRelease();
  const data = release.data;
  return <section className="bassaps-panel bassaps-platform-release" aria-label={t('platform.releaseTitle')} aria-busy={release.phase === 'loading'}>
    <div className="bassaps-panel-heading"><h2>{t('platform.releaseTitle')}</h2>
      <button type="button" className="bassaps-platform-link" disabled={release.phase === 'loading'} onClick={refreshRelease}>{t('platform.refresh')}</button></div>
    <dl className="bassaps-about-meta">
      <div><dt>{t('platform.installed')}</dt><dd>{installedVersion.data ?? t(installedVersion.phase === 'loading' ? 'platform.loading' : 'platform.installedError')}</dd></div>
      <div><dt>{t('platform.publicVersion')}</dt><dd>{data?.latest?.version ?? t(data ? 'platform.noRelease' : release.phase === 'loading' ? 'platform.loading' : 'platform.unconfirmed')}</dd></div>
    </dl>
    <div role="status" aria-live="polite">
      {release.error && <p>{t('platform.releaseError')}</p>}
      {release.expired && <p>{t('platform.expired')}</p>}
      {data && <>
        {releaseAge(data, release.now) === 'stale' && <p>{t('platform.oldRelease')}</p>}
        <p>{t('platform.checked')}: <time dateTime={data.fetchedAt} title={data.fetchedAt}>{new Date(data.fetchedAt).toLocaleString(i18n.resolvedLanguage)}</time></p>
        {data.latest && <>
          <p>{t('platform.tag')}: {data.latest.tag}</p>
          <p>{data.assetState === 'available' && data.recommendedAsset
            ? <>{data.recommendedAsset.name} · {new Intl.NumberFormat(i18n.resolvedLanguage).format(data.recommendedAsset.size)} bytes</>
            : t(data.assetState === 'none' ? 'platform.noAsset' : 'platform.noAssetInfo')}</p>
          {data.recommendedAsset?.digest && <p className="bassaps-platform-digest">{t('platform.digest')}: {data.recommendedAsset.digest}</p>}
        </>}
      </>}
    </div>
    <p>{t('platform.displayOnly')}</p>
    <div className="bassaps-platform-links"><PlatformLink id="download" /><PlatformLink id="releases" /></div>
  </section>;
}
