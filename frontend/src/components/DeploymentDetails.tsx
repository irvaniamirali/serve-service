import { useEffect, useState } from 'react'
import { RefreshCw, X } from 'lucide-react'
import { api, ApiError, type Deployment, type DeploymentBuildLog } from '@/api/client'
import { CopyValue } from '@/components/CopyValue'
import { Notice } from '@/components/Notice'
import { StatusBadge } from '@/components/StatusBadge'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardHeader, CardPanel, CardTitle } from '@/components/ui/card'

const pollInterval = 4000

interface DeploymentDetailsProps {
  deploymentId: number
  onClose: () => void
}

function errorMessage(error: unknown): string {
  return error instanceof ApiError
    ? error.message
    : 'Could not reach the API. Check the backend and try again.'
}

function formatDate(value?: string): string {
  if (!value) return 'Not available'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function buildDuration(deployment: Deployment): string {
  if (!deployment.build_started_at || !deployment.build_finished_at) return 'Not available'
  const start = new Date(deployment.build_started_at).getTime()
  const finish = new Date(deployment.build_finished_at).getTime()
  if (Number.isNaN(start) || Number.isNaN(finish) || finish < start) return 'Not available'
  const seconds = Math.round((finish - start) / 1000)
  return seconds < 60
    ? `${seconds} sec`
    : `${Math.floor(seconds / 60)} min ${seconds % 60} sec`
}

function buildIsRunning(status: string): boolean {
  return status === 'queued' || status === 'building'
}

export function DeploymentDetails({ deploymentId, onClose }: DeploymentDetailsProps) {
  const [deployment, setDeployment] = useState<Deployment | null>(null)
  const [buildLog, setBuildLog] = useState<DeploymentBuildLog | null>(null)
  const [loading, setLoading] = useState(true)
  const [deploymentError, setDeploymentError] = useState('')
  const [logError, setLogError] = useState('')
  const [refreshVersion, setRefreshVersion] = useState(0)

  useEffect(() => {
    let mounted = true
    let retryTimer: ReturnType<typeof setTimeout> | undefined
    let controller: AbortController | undefined
    let shouldPoll = false

    async function refresh() {
      controller = new AbortController()
      setDeploymentError('')
      try {
        const current = await api.getDeployment(deploymentId, controller.signal)
        if (!mounted) return
        setDeployment(current)
        setLoading(false)
        shouldPoll = buildIsRunning(current.status)

        try {
          const logs = await api.getDeploymentBuildLog(deploymentId, controller.signal)
          if (mounted) {
            setBuildLog(logs)
            setLogError('')
          }
        } catch (error) {
          if (!controller.signal.aborted && mounted) setLogError(errorMessage(error))
        }

        if (shouldPoll) {
          retryTimer = setTimeout(() => void refresh(), pollInterval)
        }
      } catch (error) {
        if (controller.signal.aborted || !mounted) return
        setDeploymentError(errorMessage(error))
        setLoading(false)
        if (shouldPoll) {
          retryTimer = setTimeout(() => void refresh(), pollInterval)
        }
      }
    }

    setDeployment(null)
    setBuildLog(null)
    setDeploymentError('')
    setLogError('')
    setLoading(true)
    void refresh()

    return () => {
      mounted = false
      if (retryTimer) clearTimeout(retryTimer)
      controller?.abort()
    }
  }, [deploymentId, refreshVersion])

  return (
    <Card className="content-card deployment-detail-card">
      <CardHeader className="section-card-header">
        <div>
          <CardTitle>Deployment #{deploymentId}</CardTitle>
          <CardDescription>Build outcome and diagnostic output from the deployment API.</CardDescription>
        </div>
        <div className="history-actions">
          <Button
            aria-label="Refresh deployment details"
            disabled={loading}
            onClick={() => setRefreshVersion((version) => version + 1)}
            size="sm"
            variant="outline"
          >
            <RefreshCw aria-hidden="true" size={13} /> Refresh
          </Button>
          <Button aria-label="Close deployment details" onClick={onClose} size="sm" variant="ghost">
            <X aria-hidden="true" size={14} /> Close
          </Button>
        </div>
      </CardHeader>
      <CardPanel>
        {loading && <p className="muted">Loading deployment details…</p>}
        {deploymentError && <Notice kind="error">{deploymentError}</Notice>}
        {deployment && (
          <>
            <div aria-live="polite" className="deployment-result-head">
              <div>
                <span className="result-label">CURRENT STATUS</span>
                <StatusBadge status={deployment.status} />
              </div>
              {buildIsRunning(deployment.status) && <span className="muted">Refreshing every 4 seconds</span>}
            </div>
            <dl className="detail-grid">
              <div className="detail-item">
                <dt>Application ID</dt>
                <dd><CopyValue label="application ID" value={String(deployment.application_id)} /></dd>
              </div>
              <div className="detail-item">
                <dt>Resolved commit</dt>
                <dd>{deployment.commit_sha ? <CopyValue compact label="commit SHA" value={deployment.commit_sha} /> : 'Not specified'}</dd>
              </div>
              <div className="detail-item">
                <dt>Build started</dt>
                <dd>{formatDate(deployment.build_started_at)}</dd>
              </div>
              <div className="detail-item">
                <dt>Build finished</dt>
                <dd>{formatDate(deployment.build_finished_at)}</dd>
              </div>
              <div className="detail-item">
                <dt>Build duration</dt>
                <dd>{buildDuration(deployment)}</dd>
              </div>
              <div className="detail-item detail-wide">
                <dt>Image reference</dt>
                <dd>{deployment.image_reference
                  ? <CopyValue compact label="image reference" value={deployment.image_reference} />
                  : 'Not available'}
                </dd>
              </div>
              <div className="detail-item detail-wide">
                <dt>Local image ID</dt>
                <dd>{deployment.image_digest
                  ? <CopyValue compact label="local image ID" value={deployment.image_digest} />
                  : 'Not available'}
                </dd>
              </div>
              {deployment.error_message && (
                <div className="detail-item detail-wide">
                  <dt>Failure details</dt>
                  <dd className="failure-message">{deployment.error_message}</dd>
                </div>
              )}
            </dl>
            <div className="build-log-section">
              <div className="build-log-heading">
                <div>
                  <h3>Build output</h3>
                  <p className="form-footnote">
                    Output is saved when the build attempt finishes; it may be empty while a build is running.
                  </p>
                </div>
              </div>
              {logError && <Notice kind="error">{logError}</Notice>}
              <pre aria-label={`Build output for deployment ${deploymentId}`} className="build-log">
                {buildLog?.log || 'No build output is available yet.'}
              </pre>
            </div>
          </>
        )}
      </CardPanel>
    </Card>
  )
}
