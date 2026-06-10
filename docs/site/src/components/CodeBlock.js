import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from 'react';
export default function CodeBlock({ code, lang }) {
    const [copied, setCopied] = useState(false);
    const copy = () => {
        navigator.clipboard.writeText(code).then(() => {
            setCopied(true);
            setTimeout(() => setCopied(false), 1500);
        });
    };
    return (_jsxs("div", { className: "code-wrap", children: [_jsxs("div", { className: "code-hdr", children: [_jsx("span", { children: lang || 'json' }), _jsx("button", { onClick: copy, children: copied ? 'Kopiert!' : 'Kopieren' })] }), _jsx("pre", { children: _jsx("code", { children: code }) })] }));
}
