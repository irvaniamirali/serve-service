import { Badge } from '@/components/ui/badge'

const statusVariants: Record<string, 'secondary' | 'info' | 'success' | 'error' | 'outline'> = {
  queued: 'secondary',
  building: 'info',
  built: 'success',
  deploying: 'info',
  healthy: 'success',
  failed: 'error',
  cancelled: 'outline',
}

export function StatusBadge({ status }: { status: string }) {
  return (
    <Badge className="status-badge" variant={statusVariants[status] ?? 'outline'}>
      <span aria-hidden="true" className={`status-dot status-${status}`} />
      {status}
    </Badge>
  )
}
