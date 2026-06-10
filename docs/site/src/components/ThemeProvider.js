import { jsx as _jsx } from "react/jsx-runtime";
import { createContext, useContext, useState, useEffect } from 'react';
const ThemeCtx = createContext({ theme: 'dark', toggle: () => { } });
export function ThemeProvider({ children }) {
    const [theme, setTheme] = useState(() => {
        const s = localStorage.getItem('gd-theme');
        return s === 'light' || s === 'dark' ? s : 'dark';
    });
    useEffect(() => {
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem('gd-theme', theme);
    }, [theme]);
    return _jsx(ThemeCtx.Provider, { value: { theme, toggle: () => setTheme(t => t === 'dark' ? 'light' : 'dark') }, children: children });
}
export const useTheme = () => useContext(ThemeCtx);
