import { useEffect, useState } from 'react'

export default function App() {
  const [backendStatus, setBackendStatus] = useState('vérification…')

  useEffect(() => {
    fetch('/api/health')
      .then((response) => {
        if (!response.ok) {
          throw new Error('Réponse invalide du serveur')
        }

        return response.json()
      })
      .then((data) => setBackendStatus(data.status))
      .catch(() => setBackendStatus('indisponible'))
  }, [])

  return (
    <main className="app">
      <h1>Simulation SMA</h1>
      <p>Interface de la simulation du réseau de bornes de recharge.</p>
      <p className="status">
        Backend : <strong>{backendStatus}</strong>
      </p>
    </main>
  )
}
