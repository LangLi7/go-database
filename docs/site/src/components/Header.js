import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useTheme } from './ThemeProvider';
export default function Header() {
    const { theme, toggle } = useTheme();
    return (_jsxs("header", { className: "header", children: [_jsxs("a", { href: "/", className: "logo", children: ["go-database ", _jsx("span", { children: "docs" })] }), _jsx("span", { className: "badge", children: "v0.1.0" }), _jsxs("nav", { className: "nav-center", children: [_jsx("a", { href: "/", children: "Home" }), _jsx("a", { href: "/api", children: "API" }), _jsx("a", { href: "/dashboard", children: "Dashboard" })] }), _jsx("button", { className: "theme-btn", onClick: toggle, "aria-label": "Theme umschalten", children: theme === 'dark' ? '\u2600' : '\u263E' })] }));
}
