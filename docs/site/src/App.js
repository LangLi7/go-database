import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { ThemeProvider } from './components/ThemeProvider';
import Header from './components/Header';
import Home from './pages/Home';
import ApiDocs from './pages/ApiDocs';
import DashboardPage from './pages/Dashboard';
import './styles/global.css';
export default function App() {
    return (_jsx(ThemeProvider, { children: _jsxs(BrowserRouter, { children: [_jsx(Header, {}), _jsxs(Routes, { children: [_jsx(Route, { path: "/", element: _jsx(Home, {}) }), _jsx(Route, { path: "/api", element: _jsx(ApiDocs, {}) }), _jsx(Route, { path: "/dashboard", element: _jsx(DashboardPage, {}) })] })] }) }));
}
