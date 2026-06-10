import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { endpoints } from '../data/endpoints';
const groups = [
    { label: 'Auth', ids: ['login', 'refresh', 'verify', 'change-pw'] },
    { label: 'Connections', ids: ['list-conn', 'create-conn', 'get-conn', 'del-conn', 'ping-conn'] },
    { label: 'Explorer', ids: ['tables', 'schema', 'browse', 'insert', 'update', 'delete'] },
    { label: 'Query', ids: ['query', 'execute', 'safe'] },
    { label: 'Datenbanken', ids: ['standalone', 'list-db', 'create-db', 'drop-db', 'create-tbl', 'drop-tbl'] },
    { label: 'Admin', ids: ['stats', 'activity', 'get-design', 'save-design', 'list-users', 'create-user', 'update-user', 'delete-user', 'get-perms', 'set-perms', 'get-db', 'set-db', 'list-roles', 'create-role', 'update-role', 'delete-role', 'role-perms', 'perm-groups'] },
    { label: 'API Keys', ids: ['list-keys', 'create-key', 'delete-key'] },
    { label: 'Transfer', ids: ['start-transfer', 'get-transfer', 'cancel-transfer', 'transfer-log'] },
    { label: 'WebSocket', ids: ['ws-query'] },
    { label: 'SSE', ids: ['sse-activity', 'sse-stats'] },
    { label: 'Weitere', ids: ['suggest', 'traffic-stats', 'traffic-reqs', 'health'] },
];
const map = new Map();
endpoints.forEach(e => map.set(e.id, e));
export default function Sidebar({ current }) {
    return (_jsx("aside", { className: "sidebar", children: groups.map(g => (_jsxs("div", { className: "sg", children: [_jsx("div", { className: "sgt", children: g.label }), g.ids.map(id => {
                    const ep = map.get(id);
                    if (!ep)
                        return null;
                    return (_jsxs("a", { href: `/api#${id}`, className: `sl ${current === id ? 'active' : ''}`, children: [_jsx("span", { className: `m method-${ep.method}`, children: ep.method }), ep.short] }, id));
                })] }, g.label))) }));
}
