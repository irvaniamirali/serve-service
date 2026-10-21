import { useEffect, useState, type FormEvent } from 'react'
import { Search } from 'lucide-react'
import { api, ApiError, type Deployment } from '@/api/client'
import { ApplicationDetails } from '@/components/ApplicationDetails'
import { CopyValue } from '@/components/CopyValue'
import { Notice } from '@/components/Notice'
import { StatusBadge } from '@/components/StatusBadge'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardHeader, CardPanel, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { parsePositiveId } from '@/lib/parse-positive-id'

interface DeploymentsPageProps {
  applicationId: number | null
  onApplicationSelected: (id: number | null) => void
}

function errorMessage(error: unknown): string {
  return error instanceof ApiError
    ? error.message
    : 'Could not reach the API. Check the backend and try again.'
}

function formatDate(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function DeploymentLookup() {
  const [lookupId, setLookupId] = useState('')
  const [deployment, setDeployment] = useState<Deployment | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function lookupDeployment(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setDeployment(null)
    const id = parsePositiveId(lookupId)
    if (id === null) {
      setError('Enter a positive deployment ID.')
      return
    }
    setLoading(true)
    try {
      setDeployment(await api.getDeployment(id))
    } catch (requestError) {
      setError(errorMessage(requestError))
    } finally {
      setLoading(false)
    }
  }

  return (
    <Card className="content-card">
      <CardHeader>
        <div className="card-heading-icon"><Search aria-hidden="true" size={16} /></div>
        <CardTitle>Retrieve a request</CardTitle>
        <CardDescription>Look up one deployment using its API-issued ID.</CardDescription>
      </CardHeader>
      <CardPanel>
        <form className="lookup-form" onSubmit={lookupDeployment}>
          <label className="field field-grow" htmlFor="deployment-lookup">
            Deployment ID
            <Input
              className="text-input mono-input"
              id="deployment-lookup"
              inputMode="numeric"
              nativeInput
              onChange={(event) => setLookupId(event.target.value)}
              placeholder="e.g. 42"
              value={lookupId}
            />
          </label>
          <Button className="lookup-submit" disabled={loading} loading={loading} type="submit">Retrieve</Button>
        </form>
        {error && <Notice kind="error">{error}</Notice>}
        {deployment && (
          <div className="deployment-result">
            <div className="deployment-result-head">
              <div>
                <span className="result-label">DEPLOYMENT</span>
                <CopyValue label="deployment ID" value={String(deployment.id)} />
              </div>
              <StatusBadge status={deployment.status} />
            </div>
            <dl className="detail-grid">
              <div className="detail-item">
                <dt>Application ID</dt>
                <dd><CopyValue label="application ID" value={String(deployment.application_id)} /></dd>
              </div>
              <div className="detail-item">
                <dt>Commit SHA</dt>
                <dd>{deployment.commit_sha ? <CopyValue compact label="commit SHA" value={deployment.commit_sha} /> : 'Not specified'}</dd>
              </div>
              <div className="detail-item">
                <dt>Created</dt>
                <dd>{formatDate(deployment.created_at)}</dd>
              </div>
              <div className="detail-item">
                <dt>Updated</dt>
                <dd>{formatDate(deployment.updated_at)}</dd>
              </div>
              {deployment.error_message && (
                <div className="detail-item detail-wide">
                  <dt>Failure details</dt>
                  <dd className="failure-message">{deployment.error_message}</dd>
                </div>
              )}
            </dl>
          </div>
        )}
      </CardPanel>
    </Card>
  )
}

export function DeploymentsPage({
  applicationId,
  onApplicationSelected,
}: DeploymentsPageProps) {
  const [lookupId, setLookupId] = useState(applicationId ? String(applicationId) : '')
  const [lookupError, setLookupError] = useState('')

  useEffect(() => {
    setLookupId(applicationId ? String(applicationId) : '')
  }, [applicationId])

  function selectApplication(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setLookupError('')
    const id = parsePositiveId(lookupId)
    if (id === null) {
      setLookupError('Enter a positive application ID.')
      return
    }
    onApplicationSelected(id)
  }

  return (
    <section aria-labelledby="deployments-title">
      <div className="page-heading">
        <div>
          <p className="eyebrow">REQUEST HISTORY</p>
          <h1 id="deployments-title">Deployments</h1>
          <p className="page-subtitle">
            Inspect recorded requests. The current API does not expose a global deployment list.
          </p>
        </div>
      </div>

      <div className="deployment-tools">
        <Card className="content-card">
          <CardHeader>
            <CardTitle>Application history</CardTitle>
            <CardDescription>Load the most recent deployment requests for an application.</CardDescription>
          </CardHeader>
          <CardPanel>
            <form className="lookup-form" onSubmit={selectApplication}>
              <label className="field field-grow" htmlFor="history-application-id">
                Application ID
                <Input
                  className="text-input mono-input"
                  id="history-application-id"
                  inputMode="numeric"
                  nativeInput
                  onChange={(event) => setLookupId(event.target.value)}
                  placeholder="e.g. 12"
                  value={lookupId}
                />
              </label>
              <Button className="lookup-submit" variant="outline" type="submit">Load history</Button>
            </form>
            {lookupError && <Notice kind="error">{lookupError}</Notice>}
            <p className="form-footnote">History is paginated by the API and this view loads the latest 20 entries.</p>
          </CardPanel>
        </Card>
        <DeploymentLookup />
      </div>

      {applicationId !== null && (
        <div className="application-workspace">
          <ApplicationDetails
            applicationId={applicationId}
            onApplicationUnavailable={() => onApplicationSelected(null)}
          />
        </div>
      )}
    </section>
  )
}
