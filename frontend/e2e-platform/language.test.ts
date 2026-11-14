import { test } from 'node:test';
import { strict as assert } from 'node:assert';
import { normalizeLanguage } from '../src/lib/language.ts';

test('supported languages and existing regional identifiers keep their language', () => {
  for (const [input, expected] of [['en', 'en'], ['JA', 'ja'], ['ja-JP', 'ja'], ['zh-Hant-TW', 'zh'], ['ko_KR', 'ko'], ['vi-VN', 'vi']]) {
    assert.equal(normalizeLanguage(input), expected);
  }
});

test('untrusted stored language values fail closed to the existing English fallback', () => {
  for (const input of [null, undefined, 1, true, {}, [], '', 'fr', '../', '%2e%2e%2f', 'https://fixture.invalid', 'file:///fixture', 'data:text/plain,ja', 'ja&ns=x', 'ja/../../', 'ja-' + 'x'.repeat(100_000)]) {
    assert.equal(normalizeLanguage(input), 'en');
  }
});
