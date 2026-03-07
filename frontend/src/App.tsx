import { useEffect, useState } from 'react'

function App() {
  const [status, setStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting')

  useEffect(() => {
    fetch('/-/api/status')
      .then((res) => {
        if (res.ok) setStatus('connected')
        else setStatus('disconnected')
      })
      .catch(() => setStatus('disconnected'))
  }, [])

  return (
    <div style={{ fontFamily: 'system-ui, sans-serif', padding: '2rem' }}>
      <h1>crelay dashboard</h1>
      <p>
        Status:{' '}
        <span style={{ color: status === 'connected' ? '#22c55e' : status === 'connecting' ? '#eab308' : '#ef4444' }}>
          {status}
        </span>
      </p>
    </div>
  )
}

export default App
