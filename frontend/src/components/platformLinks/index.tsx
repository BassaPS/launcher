import { AppService } from '@bindings/cyrene-launcher/internal/app-service';
import { useTranslation } from 'react-i18next';
import { toast } from 'react-toastify';
import { platformLinks, serviceLinkIDs } from '@/lib/platform/links';
import '@/styles/platform.css';

export function PlatformLink({ id }: { id: keyof typeof platformLinks }) {
  const { t } = useTranslation();
  return <button type="button" className="bassaps-platform-link" onClick={() => {
    void AppService.OpenURL(platformLinks[id]).catch(() => toast.error(t('platform.linkError')));
  }}>{t(`platform.links.${id}`)} <span aria-hidden="true">↗</span></button>;
}

export default function PlatformLinks() {
  const { t } = useTranslation();
  return <nav className="bassaps-platform-links" aria-label={t('platform.services')}>
    {serviceLinkIDs.map(id => <PlatformLink key={id} id={id} />)}
  </nav>;
}
