import { useState, useMemo } from 'react'
import { endpoints } from '../data/endpoints'

export default function Search() {
  const [q, setQ] = useState('')
  const [open, setOpen] = useState(false)

  const results = useMemo(() => {
    if (!q.trim()) return []
    const low = q.toLowerCase()
    return endpoints.filter(e =>
      e.path.toLowerCase().includes(low) ||
      e.desc.toLowerCase().includes(low) ||
      e.group.toLowerCase().includes(low)
    ).slice(0, 12)
  }, [q])

  return (
    <div className="search-wrap">
      <input
        className="search-input"
        placeholder="Suche Endpunkte..."
        value={q}
        onChange={e => { setQ(e.target.value); setOpen(true) }}
        onFocus={() => setOpen(true)}
        onBlur={() => setTimeout(() => setOpen(false), 200)}
      />
      {open && results.length > 0 && (
        <div className="search-results">
          {results.map(e => (
            <a key={e.id} className="search-item" href={`/api#${e.id}`} onClick={() => { setOpen(false); setQ('') }}>
              <span className={`method method-${e.method}`}>{e.method}</span>
              <span className="path">{e.path}</span>
              <span className="desc">{e.desc}</span>
            </a>
          ))}
        </div>
      )}
    </div>
  )
}
