import { useState } from 'react'
import { Activity, Boxes, Code2, Layers3 } from 'lucide-react'
import { ApplicationsPage } from '@/pages/ApplicationsPage'
import { DeploymentsPage } from '@/pages/DeploymentsPage'
import './App.css'

type Section = 'applications' | 'deployments'

function App() {
  const [section, setSection] = useState<Section>('applications')
  const [applicationId, setApplicationId] = useState<number | null>(null)

  return (
    <div className="app-shell">
      <aside aria-label="Main navigation" className="sidebar">
        <a aria-label="Serve Service home" className="brand" href="#main">
          <span aria-hidden="true" className="brand-mark"><Layers3 size={19} /></span>
          <span>
            <strong>serve service</strong>
            <small>control plane</small>
          </span>
        </a>

        <div className="nav-label">WORKSPACE</div>
        <nav className="primary-nav">
          <button
            aria-current={section === 'applications' ? 'page' : undefined}
            className={`nav-link ${section === 'applications' ? 'nav-active' : ''}`}
            onClick={() => setSection('applications')}
            type="button"
          >
            <Boxes aria-hidden="true" size={17} />
            Applications
          </button>
          <button
            aria-current={section === 'deployments' ? 'page' : undefined}
            className={`nav-link ${section === 'deployments' ? 'nav-active' : ''}`}
            onClick={() => setSection('deployments')}
            type="button"
          >
            <Activity aria-hidden="true" size={17} />
            Deployments
          </button>
        </nav>

        <div className="sidebar-bottom">
          <div className="milestone-card">
            <div className="milestone-top">M02 · Core API</div>
            <p>Requests are recorded. Build and runtime execution are not implemented yet.</p>
          </div>
          <a className="github-link" href="https://github.com/irvaniamirali/serve-service" rel="noreferrer" target="_blank">
            <Code2 aria-hidden="true" size={15} /> Source repository
          </a>
        </div>
      </aside>

      <main className="main-area" id="main">
        <header className="topbar">
          <div className="breadcrumb"><span>Serve Service</span><span aria-hidden="true">/</span><strong>{section}</strong></div>
          <span className="api-mode">API routes · /api/v1</span>
        </header>

        <div className="page-content">
          {section === 'applications' ? (
            <ApplicationsPage
              applicationId={applicationId}
              onApplicationSelected={setApplicationId}
            />
          ) : (
            <DeploymentsPage
              applicationId={applicationId}
              onApplicationSelected={setApplicationId}
            />
          )}
        </div>
      </main>
    </div>
  )
}

export default App
