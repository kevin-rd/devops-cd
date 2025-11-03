import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import zh from './zh'
import en from './en'

const resources = {
  zh,
  en,
}

i18n.use(initReactI18next).init({
  resources,
  lng: localStorage.getItem('language') || 'zh', // 默认中文
  fallbackLng: 'zh',
  interpolation: {
    escapeValue: false,
  },
})

export default i18n

