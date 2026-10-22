import { useEffect, useState } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { api, type Application, type Deployment, ApiError } from '@/api/client'
import { CopyValue } from '@/components/CopyValue'
import { DeploymentDetails } from '@/components/DeploymentDetails'
import { Notice } from '@/components/Notice'
import { StatusBadge } from '@/components/StatusBadge'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardDescription,
  CardHeader,
  CardPanel,
  CardTitle,
} from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Input } from '@/components/ui/input'

const historyPageSize = 20

interface ApplicationDetailsProps {
  applicationId: number
  onApplicationUnavailable?: () => void
}

function formatDate(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function getErrorMessage(error: unknown): string {
  return error instanceof ApiError ? error.message : 'Could not reach the API. Check the backend and try again.'
}

export function ApplicationDetails({
  applicationId,
  onApplicationUnavailable,
}: ApplicationDetailsProps) {
  const [application, setApplication] = useState<Application | null>(null)
  const [applicationError, setApplicationError] = useState('')
  const [applicationLoading, setApplicationLoading] = useState(true)
  const [deployments, setDeployments] = useState<Deployment[]>([])
  const [historyOffset, setHistoryOffset] = useState(0)
  const [historyCanLoadOlder, setHistoryCanLoadOlder] = useState(false)
  const [historyError, setHistoryError] = useState('')
  const [historyLoading, setHistoryLoading] = useState(true)
  const [commitSHA, setCommitSHA] = useState('')
  const [creatingDeployment, setCreatingDeployment] = useState(false)
  const [createdDeployment, setCreatedDeployment] = useState<Deployment | null>(null)
  const [selectedDeploymentID, setSelectedDeploymentID] = useState<number | null>(null)
  const [createError, setCreateError] = useState('')

  async function refreshHistory(signal?: AbortSignal, offset = historyOffset) {
    setHistoryLoading(true)
    setHistoryError('')
    try {
      const history = await api.listDeployments(
        applicationId,
        historyPageSize,
        offset,
        signal,
      )
      setDeployments(history.deployments)
      setHistoryOffset(offset)
      setHistoryCanLoadOlder(history.deployments.length === historyPageSize)
    } catch (error) {
      if (signal?.aborted) return
      setHistoryError(getErrorMessage(error))
    } finally {
      if (!signal?.aborted) setHistoryLoading(false)
    }
  }

  useEffect(() => {
    const controller = new AbortController()
    setApplication(null)
    setApplicationError('')
    setApplicationLoading(true)
    setDeployments([])
    setHistoryOffset(0)
    setHistoryCanLoadOlder(false)
    setHistoryLoading(true)
    setHistoryError('')
    setCreatedDeployment(null)
    setSelectedDeploymentID(null)

    api.getApplication(applicationId, controller.signal)
      .then((result) => setApplication(result))
      .catch((error: unknown) => {
        if (controller.signal.aborted) return
        setApplicationError(getErrorMessage(error))
        if (error instanceof ApiError && error.status === 404) {
          onApplicationUnavailable?.()
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setApplicationLoading(false)
      })

    void refreshHistory(controller.signal, 0)
    return () => controller.abort()
  }, [applicationId])

  async function handleCreateDeployment(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setCreatingDeployment(true)
    setCreateError('')
    setCreatedDeployment(null)
    try {
      const created = await api.createDeployment(applicationId, commitSHA.trim())
      setCreatedDeployment(created)
      setSelectedDeploymentID(created.id)
      setCommitSHA('')
      await refreshHistory(undefined, 0)
    } catch (error) {
      setCreateError(getErrorMessage(error))
    } finally {
      setCreatingDeployment(false)
    }
  }

  if (applicationLoading) {
    return <Card className="content-card"><CardPanel><p className="muted">Loading application…</p></CardPanel></Card>
  }

  if (applicationError || !application) {
    return (
      <Card className="content-card">
        <CardPanel>
          <Notice kind="error">{applicationError || 'Application details are unavailable.'}</Notice>
        </CardPanel>
      </Card>
    )
  }

  return (
    <div className="details-stack">
      <Card className="content-card">
        <CardHeader className="section-card-header">
          <div>
            <CardTitle>{application.name}</CardTitle>
            <CardDescription>Application configuration returned by the API</CardDescription>
          </div>
          <Badge variant="outline">Application</Badge>
        </CardHeader>
        <CardPanel>
          <dl className="detail-grid">
            <div className="detail-item">
              <dt>Application ID</dt>
              <dd><CopyValue label="application ID" value={String(application.id)} /></dd>
            </div>
            <div className="detail-item">
              <dt>Branch</dt>
              <dd><code>{application.branch}</code></dd>
            </div>
            <div className="detail-item">
              <dt>GitHub webhook</dt>
              <dd>{application.github_repository_id
                ? `Repository #${application.github_repository_id}`
                : 'Not linked'}
              </dd>
            </div>
            <div className="detail-item detail-wide">
              <dt>Repository</dt>
              <dd><a href={application.repository_url} rel="noreferrer" target="_blank">{application.repository_url}</a></dd>
            </div>
            <div className="detail-item">
              <dt>Created</dt>
              <dd>{formatDate(application.created_at)}</dd>
            </div>
            <div className="detail-item">
              <dt>Updated</dt>
              <dd>{formatDate(application.updated_at)}</dd>
            </div>
          </dl>
        </CardPanel>
      </Card>

      <Card className="content-card">
        <CardHeader>
          <CardTitle>Request a deployment</CardTitle>
          <CardDescription>
            Leave the commit blank to use the latest commit on GitHub&apos;s default branch, or provide a full SHA to build that exact revision.
          </CardDescription>
        </CardHeader>
        <CardPanel>
          <form className="deployment-form" onSubmit={handleCreateDeployment}>
            <label className="field field-grow" htmlFor={`commit-${application.id}`}>
              Commit SHA <span className="field-hint">Optional · full 40- or 64-character Git SHA</span>
              <Input
                autoComplete="off"
                className="text-input mono-input"
                id={`commit-${application.id}`}
                maxLength={64}
                nativeInput
                onChange={(event) => setCommitSHA(event.target.value)}
                pattern="[a-fA-F0-9]{40}|[a-fA-F0-9]{64}"
                placeholder="Default branch latest, or enter a SHA"
                value={commitSHA}
              />
            </label>
            <Button disabled={creatingDeployment} loading={creatingDeployment} type="submit">
              Queue request
            </Button>
          </form>
          {createError && <Notice kind="error">{createError}</Notice>}
          {createdDeployment && (
            <Notice kind="success">
              Deployment #{createdDeployment.id} was created for commit{' '}
              {createdDeployment.commit_sha
                ? <CopyValue compact label="resolved commit SHA" value={createdDeployment.commit_sha} />
                : 'unresolved'}{' '}
              with status <strong>{createdDeployment.status}</strong>. Its build will start when the worker claims it.
            </Notice>
          )}
        </CardPanel>
      </Card>

      <Card className="content-card">
        <CardHeader className="section-card-header">
          <div>
            <CardTitle>Deployment history</CardTitle>
            <CardDescription>Newest first · up to 20 requests from the API</CardDescription>
          </div>
          <div className="history-actions">
            <span className="history-page">Page {Math.floor(historyOffset / historyPageSize) + 1}</span>
            <Button
              aria-label="Load newer deployment history"
              disabled={historyLoading || historyOffset === 0}
              onClick={() => void refreshHistory(undefined, Math.max(0, historyOffset - historyPageSize))}
              size="sm"
              variant="outline"
            >
              <ChevronLeft aria-hidden="true" size={14} /> Newer
            </Button>
            <Button
              aria-label="Refresh deployment history"
              disabled={historyLoading}
              onClick={() => void refreshHistory()}
              size="sm"
              variant="outline"
            >
              Refresh
            </Button>
            <Button
              aria-label="Load older deployment history"
              disabled={historyLoading || !historyCanLoadOlder}
              onClick={() => void refreshHistory(undefined, historyOffset + historyPageSize)}
              size="sm"
              variant="outline"
            >
              Older <ChevronRight aria-hidden="true" size={14} />
            </Button>
          </div>
        </CardHeader>
        <CardPanel>
          {historyLoading && <p className="muted">Loading deployment history…</p>}
          {historyError && <Notice kind="error">{historyError}</Notice>}
          {!historyLoading && !historyError && deployments.length === 0 && (
            <div className="empty-state">
              {historyOffset === 0 ? (
                <>
                  <p>No deployment requests for this application yet.</p>
                  <span>Use “Queue request” above to create the first request.</span>
                </>
              ) : (
                <>
                  <p>No more deployment requests on this page.</p>
                  <span>Use “Newer” to return to the latest history.</span>
                </>
              )}
            </div>
          )}
          {deployments.length > 0 && (
            <div className="table-wrap">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Request</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Commit</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead>Image</TableHead>
                    <TableHead>Failure details</TableHead>
                    <TableHead>Details</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {deployments.map((deployment) => (
                    <TableRow key={deployment.id}>
                      <TableCell><CopyValue compact label="deployment ID" value={String(deployment.id)} /></TableCell>
                      <TableCell><StatusBadge status={deployment.status} /></TableCell>
                      <TableCell>
                        {deployment.commit_sha
                          ? <CopyValue compact label="commit SHA" value={deployment.commit_sha} />
                          : <span className="muted">Not specified</span>}
                      </TableCell>
                      <TableCell>{formatDate(deployment.created_at)}</TableCell>
                      <TableCell className="failure-cell">
                        {deployment.image_reference || <span className="muted">Not built</span>}
                      </TableCell>
                      <TableCell className="failure-cell">
                        {deployment.error_message || <span className="muted">—</span>}
                      </TableCell>
                      <TableCell>
                        <Button
                          aria-label={`Open deployment ${deployment.id} details`}
                          onClick={() => setSelectedDeploymentID(deployment.id)}
                          size="sm"
                          variant="outline"
                        >
                          Details
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardPanel>
      </Card>
      {selectedDeploymentID !== null && (
        <DeploymentDetails
          deploymentId={selectedDeploymentID}
          onClose={() => setSelectedDeploymentID(null)}
        />
      )}
    </div>
  )
}
