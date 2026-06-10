import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { ThemeProvider } from './components/ThemeProvider'
import Header from './components/Header'
import Home from './pages/Home'
import ApiDocs from './pages/ApiDocs'
import DashboardPage from './pages/Dashboard'
import './styles/global.css'

export default function App() {
  return (
    <ThemeProvider>
      <BrowserRouter>
        <Header />
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/api" element={<ApiDocs />} />
          <Route path="/dashboard" element={<DashboardPage />} />
        </Routes>
      </BrowserRouter>
    </ThemeProvider>
  )
}
