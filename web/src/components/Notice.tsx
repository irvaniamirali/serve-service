interface NoticeProps {
  children: React.ReactNode
  kind?: 'error' | 'info' | 'success'
}

export function Notice({ children, kind = 'info' }: NoticeProps) {
  return (
    <div className={`notice notice-${kind}`} role={kind === 'error' ? 'alert' : 'status'}>
      {children}
    </div>
  )
}
