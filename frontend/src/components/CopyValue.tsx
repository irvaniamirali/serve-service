import { useState } from 'react'
import { Button } from '@/components/ui/button'

interface CopyValueProps {
  value: string
  label: string
  compact?: boolean
}

export function CopyValue({ value, label, compact = false }: CopyValueProps) {
  const [message, setMessage] = useState('')

  async function copyValue() {
    try {
      await navigator.clipboard.writeText(value)
      setMessage('Copied')
    } catch {
      setMessage('Copy unavailable')
    }
  }

  return (
    <span className="copy-value">
      <code className={compact ? 'mono truncate' : 'mono'} title={value}>
        {value}
      </code>
      <Button
        aria-label={`Copy ${label}`}
        className="copy-button"
        onClick={copyValue}
        size="xs"
        variant="ghost"
      >
        {message || 'Copy'}
      </Button>
      <span className="sr-only" role="status">{message}</span>
    </span>
  )
}
