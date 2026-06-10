import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState, useMemo } from 'react';
import { endpoints } from '../data/endpoints';
export default function Search() {
    const [q, setQ] = useState('');
    const [open, setOpen] = useState(false);
    const results = useMemo(() => {
        if (!q.trim())
            return [];
        const low = q.toLowerCase();
        return endpoints.filter(e => e.path.toLowerCase().includes(low) ||
            e.desc.toLowerCase().includes(low) ||
            e.group.toLowerCase().includes(low)).slice(0, 12);
    }, [q]);
    return (_jsxs("div", { className: "search-wrap", children: [_jsx("input", { className: "search-input", placeholder: "Suche Endpunkte...", value: q, onChange: e => { setQ(e.target.value); setOpen(true); }, onFocus: () => setOpen(true), onBlur: () => setTimeout(() => setOpen(false), 200) }), open && results.length > 0 && (_jsx("div", { className: "search-results", children: results.map(e => (_jsxs("a", { className: "search-item", href: `/api#${e.id}`, onClick: () => { setOpen(false); setQ(''); }, children: [_jsx("span", { className: `method method-${e.method}`, children: e.method }), _jsx("span", { className: "path", children: e.path }), _jsx("span", { className: "desc", children: e.desc })] }, e.id))) }))] }));
}
