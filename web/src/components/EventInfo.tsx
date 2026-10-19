import type { Event } from '../api'
import { formatAddress } from '../format'

/** Ort mit Adresse und verantwortlicher Projektleiter eines Events. */
export default function EventInfo({ event }: { event: Event }) {
  const address = formatAddress(event.location)
  return (
    <p className="muted">
      {event.location.name}
      {address && `, ${address}`}
      <br />
      Projektleitung: {event.leader?.name || '–'}
    </p>
  )
}
