import { useState } from 'react'

export default function CodeBlock({ code, lang }: { code: string; lang?: string }) {
  const [copied, setCopied] = useState(false)

  const copy = () => {
    navigator.clipboard.writeText(code).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    })
  }

  return (
    <div className="code-wrap">
      <div className="code-hdr">
        <span>{lang || 'json'}</span>
        <button onClick={copy}>{copied ? 'Kopiert!' : 'Kopieren'}</button>
      </div>
      <pre><code>{code}</code></pre>
    </div>
  )
}
