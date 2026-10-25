import { useEffect, useState, type FormEvent } from 'react'
import { ArrowRight, ChevronLeft, ChevronRight, Plus, Search } from 'lucide-react'
import { api, ApiError, type Application } from '@/api/client'
import { ApplicationDetails } from '@/components/ApplicationDetails'
import { Notice } from '@/components/Notice'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardHeader, CardPanel, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { parsePositiveId } from '@/lib/parse-positive-id'

const applicationPageSize = 20

interface ApplicationsPageProps {
  applicationId: number | null
  onApplicationSelected: (id: number | null) => void
}

function errorMessage(error: unknown): string {
  return error instanceof ApiError
    ? error.message
    : 'Could not reach the API. Check the backend and try again.'
}

export function ApplicationsPage({
  applicationId,
  onApplicationSelected,
}: ApplicationsPageProps) {
  const [lookupId, setLookupId] = useState(applicationId ? String(applicationId) : '')
  const [name, setName] = useState('')
  const [repositoryURL, setRepositoryURL] = useState('')
  const [branch, setBranch] = useState('main')
  const [githubRepositoryID, setGithubRepositoryID] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [lookupError, setLookupError] = useState('')
  const [createError, setCreateError] = useState('')
  const [applications, setApplications] = useState<Application[]>([])
  const [applicationsOffset, setApplicationsOffset] = useState(0)
  const [applicationsLoading, setApplicationsLoading] = useState(true)
  const [applicationsError, setApplicationsError] = useState('')

  async function refreshApplications(signal?: AbortSignal, offset = applicationsOffset) {
    setApplicationsLoading(true)
    setApplicationsError('')
    try {
      const result = await api.listApplications(applicationPageSize, offset, signal)
      setApplications(result.applications)
      setApplicationsOffset(result.offset)
    } catch (error) {
      if (signal?.aborted) return
      setApplicationsError(errorMessage(error))
    } finally {
      if (!signal?.aborted) setApplicationsLoading(false)
    }
  }

  useEffect(() => {
    const controller = new AbortController()
    void refreshApplications(controller.signal, 0)
    return () => controller.abort()
  }, [])

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

  async function createApplication(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const parsedRepositoryID = githubRepositoryID.trim()
      ? parsePositiveId(githubRepositoryID)
      : null
    if (githubRepositoryID.trim() && parsedRepositoryID === null) {
      setCreateError('GitHub repository ID must be a positive integer.')
      return
    }
    setSubmitting(true)
    setCreateError('')
    try {
      const created = await api.createApplication({
        name,
        repository_url: repositoryURL,
        branch,
        ...(parsedRepositoryID !== null ? { github_repository_id: parsedRepositoryID } : {}),
      })
      setLookupId(String(created.id))
      onApplicationSelected(created.id)
      setName('')
      setRepositoryURL('')
      setBranch('main')
      setGithubRepositoryID('')
      await refreshApplications(undefined, 0)
    } catch (error) {
      setCreateError(errorMessage(error))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <section aria-labelledby="applications-title">
      <div className="page-heading">
        <div>
          <p className="eyebrow">SOURCE CONFIGURATION</p>
          <h1 id="applications-title">Applications</h1>
          <p className="page-subtitle">
            Register a repository or retrieve an application by its API-issued ID.
          </p>
        </div>
        {applicationId && (
          <span className="context-pill">Application #{applicationId} selected</span>
        )}
      </div>

      <div className="feature-grid">
        <Card className="content-card">
          <CardHeader>
            <div className="card-heading-icon"><Plus aria-hidden="true" size={16} /></div>
            <CardTitle>Register application</CardTitle>
            <CardDescription>Create a record with the repository and branch configuration.</CardDescription>
          </CardHeader>
          <CardPanel>
            <form className="form-stack" onSubmit={createApplication}>
              <label className="field" htmlFor="application-name">
                Application name
                <Input
                  autoComplete="off"
                  className="text-input"
                  id="application-name"
                  nativeInput
                  onChange={(event) => setName(event.target.value)}
                  placeholder="storefront"
                  required
                  value={name}
                />
              </label>
              <label className="field" htmlFor="repository-url">
                Repository URL
                <Input
                  autoComplete="url"
                  className="text-input"
                  id="repository-url"
                  nativeInput
                  onChange={(event) => setRepositoryURL(event.target.value)}
                  placeholder="https://github.com/acme/storefront.git"
                  required
                  type="url"
                  value={repositoryURL}
                />
              </label>
              <label className="field" htmlFor="application-branch">
                Branch
                <Input
                  autoComplete="off"
                  className="text-input"
                  id="application-branch"
                  nativeInput
                  onChange={(event) => setBranch(event.target.value)}
                  placeholder="main"
                  required
                  value={branch}
                />
              </label>
              <label className="field" htmlFor="github-repository-id">
                GitHub repository ID <span className="field-hint">Optional · enables webhook matching</span>
                <Input
                  autoComplete="off"
                  className="text-input mono-input"
                  id="github-repository-id"
                  inputMode="numeric"
                  min="1"
                  nativeInput
                  onChange={(event) => setGithubRepositoryID(event.target.value)}
                  placeholder="Numeric repository ID"
                  type="number"
                  value={githubRepositoryID}
                />
              </label>
              {createError && <Notice kind="error">{createError}</Notice>}
              <Button className="form-submit" disabled={submitting} loading={submitting} type="submit">
                Register application <ArrowRight aria-hidden="true" size={15} />
              </Button>
            </form>
          </CardPanel>
        </Card>

        <Card className="content-card">
          <CardHeader>
            <div className="card-heading-icon"><Search aria-hidden="true" size={16} /></div>
            <CardTitle>Open an application</CardTitle>
            <CardDescription>Retrieve a saved application by its database ID.</CardDescription>
          </CardHeader>
          <CardPanel>
            <form className="lookup-form" onSubmit={selectApplication}>
              <label className="field field-grow" htmlFor="application-lookup">
                Application ID
                <Input
                  className="text-input mono-input"
                  id="application-lookup"
                  inputMode="numeric"
                  nativeInput
                  onChange={(event) => setLookupId(event.target.value)}
                  placeholder="e.g. 12"
                  value={lookupId}
                />
              </label>
              <Button className="lookup-submit" variant="outline" type="submit">Retrieve</Button>
            </form>
            {lookupError && <Notice kind="error">{lookupError}</Notice>}
            <div className="api-gap">
              <span className="gap-indicator" aria-hidden="true">i</span>
              <p>
                Registered repositories are loaded from the API. GitHub push matching requires the numeric
                repository ID and a configured server webhook secret; no credential is entered here.
              </p>
            </div>
          </CardPanel>
        </Card>
      </div>

      <div className="application-workspace">
        <Card className="content-card">
          <CardHeader className="section-card-header">
            <div>
              <CardTitle>Registered repositories</CardTitle>
              <CardDescription>Newest first · select an application to inspect its deployments.</CardDescription>
            </div>
            <div className="history-actions">
              <Button
                aria-label="Load newer applications"
                disabled={applicationsLoading || applicationsOffset === 0}
                onClick={() => void refreshApplications(undefined, Math.max(0, applicationsOffset - applicationPageSize))}
                size="sm"
                variant="outline"
              >
                <ChevronLeft aria-hidden="true" size={14} /> Newer
              </Button>
              <Button
                aria-label="Refresh registered applications"
                disabled={applicationsLoading}
                onClick={() => void refreshApplications()}
                size="sm"
                variant="outline"
              >
                Refresh
              </Button>
              <Button
                aria-label="Load older applications"
                disabled={applicationsLoading || applications.length < applicationPageSize}
                onClick={() => void refreshApplications(undefined, applicationsOffset + applicationPageSize)}
                size="sm"
                variant="outline"
              >
                Older <ChevronRight aria-hidden="true" size={14} />
              </Button>
            </div>
          </CardHeader>
          <CardPanel>
            {applicationsLoading && <p className="muted">Loading registered repositories…</p>}
            {applicationsError && <Notice kind="error">{applicationsError}</Notice>}
            {!applicationsLoading && !applicationsError && applications.length === 0 && (
              <div className="empty-state">
                <p>No repositories registered yet.</p>
                <span>Register an application above to start requesting builds.</span>
              </div>
            )}
            {!applicationsLoading && applications.length > 0 && (
              <div className="table-wrap">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Application</TableHead>
                      <TableHead>Repository</TableHead>
                      <TableHead>Branch</TableHead>
                      <TableHead>GitHub ID</TableHead>
                      <TableHead>Action</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {applications.map((application) => (
                      <TableRow key={application.id}>
                        <TableCell>{application.name}</TableCell>
                        <TableCell className="failure-cell">{application.repository_url}</TableCell>
                        <TableCell><code>{application.branch}</code></TableCell>
                        <TableCell>{application.github_repository_id ?? <span className="muted">Not linked</span>}</TableCell>
                        <TableCell>
                          <Button
                            aria-pressed={application.id === applicationId}
                            onClick={() => {
                              setLookupId(String(application.id))
                              onApplicationSelected(application.id)
                            }}
                            size="sm"
                            variant={application.id === applicationId ? 'secondary' : 'outline'}
                          >
                            {application.id === applicationId ? 'Selected' : 'Open'}
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
