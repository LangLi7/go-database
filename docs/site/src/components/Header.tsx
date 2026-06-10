import { useTheme } from './ThemeProvider'

export default function Header() {
  const { theme, toggle } = useTheme()

  return (
    <header className="header">
      <a href="/" className="logo">go-database <span>docs</span></a>
      <span className="badge">v0.1.0</span>
      <nav className="nav-center">
        <a href="/">Home</a>
        <a href="/api">API</a>
        <a href="/dashboard">Dashboard</a>
      </nav>
      <button className="theme-btn" onClick={toggle} aria-label="Theme umschalten">
        {theme === 'dark' ? '\u2600' : '\u263E'}
      </button>
    </header>
  )
}
