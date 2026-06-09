import { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { ConfigProvider, theme } from 'antd'
import { setLang, getLang, Lang } from '../i18n/translations'

interface AppContextType {
  darkMode: boolean
  toggleDarkMode: () => void
  lang: Lang
  setLanguage: (l: Lang) => void
}

const AppContext = createContext<AppContextType>({
  darkMode: false,
  toggleDarkMode: () => {},
  lang: 'en',
  setLanguage: () => {},
})

export function useAppContext() {
  return useContext(AppContext)
}

export function AppProvider({ children }: { children: ReactNode }) {
  const [darkMode, setDarkMode] = useState(() => localStorage.getItem('darkMode') === 'true')
  const [lang, setLangState] = useState<Lang>(getLang())

  const toggleDarkMode = () => {
    setDarkMode(prev => {
      localStorage.setItem('darkMode', String(!prev))
      return !prev
    })
  }

  const setLanguage = (l: Lang) => {
    setLang(l)
    setLangState(l)
  }

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', darkMode ? 'dark' : 'light')
  }, [darkMode])

  return (
    <AppContext.Provider value={{ darkMode, toggleDarkMode, lang, setLanguage }}>
      <ConfigProvider theme={{
        algorithm: darkMode ? theme.darkAlgorithm : theme.defaultAlgorithm,
        token: {
          colorPrimary: '#6366f1',
          borderRadius: 8,
        },
      }}>
        {children}
      </ConfigProvider>
    </AppContext.Provider>
  )
}
