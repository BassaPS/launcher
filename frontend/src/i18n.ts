import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import { normalizeLanguage, supportedLanguages } from "./lib/language";
import en from "../public/locales/en.json";
import ja from "../public/locales/ja.json";
import zh from "../public/locales/zh.json";
import ko from "../public/locales/ko.json";
import vi from "../public/locales/vi.json";

// All translations ship with the Launcher. No language or namespace becomes a URL.
i18n.use(LanguageDetector).use(initReactI18next).init({
  resources: { en: { translation: en }, ja: { translation: ja }, zh: { translation: zh }, ko: { translation: ko }, vi: { translation: vi } },
  supportedLngs: [...supportedLanguages],
  load: "languageOnly",
  fallbackLng: "en",
  ns: ["translation"],
  defaultNS: "translation",
  detection: { convertDetectedLanguage: normalizeLanguage },
  interpolation: { escapeValue: false },
});
