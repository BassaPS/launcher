export const supportedLanguages = ['en', 'ja', 'zh', 'ko', 'vi'] as const;
export type Language = typeof supportedLanguages[number];

// Only bundled language identifiers can reach i18next. Storage schema is unchanged.
export function normalizeLanguage(value: unknown): Language {
  if (typeof value !== 'string' || value.length > 35) return 'en';
  const match = /^(en|ja|zh|ko|vi)(?:[-_][a-z0-9]{1,8})*$/i.exec(value);
  return match ? match[1].toLowerCase() as Language : 'en';
}
