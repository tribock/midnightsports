import type { UserStatus } from '../api'
import { statusLabels } from '../format'

/**
 * Status im Projekt (Herkunft). Mitarbeiter vergeben nur Projekt-Admins unter Teilnehmer (canSetStaff, zusammen
 * mit der Funktion), alle anderen wählen zwischen Schüler/in aus der Region und Gast extern.
 */
export default function StatusSelect({
  value,
  onChange,
  region,
  canSetStaff = false,
  disabled = false,
  label = 'Status',
}: {
  value: UserStatus | ''
  onChange: (value: UserStatus) => void
  region: string
  canSetStaff?: boolean
  disabled?: boolean
  label?: string
}) {
  const labels = statusLabels(region)
  const options = (Object.keys(labels) as UserStatus[]).filter((s) => canSetStaff || s !== 'mitarbeiter')
  return (
    <label>
      {label}
      <select required value={value} disabled={disabled} onChange={(e) => onChange(e.target.value as UserStatus)}>
        <option value="" disabled>
          Bitte wählen
        </option>
        {options.map((s) => (
          <option key={s} value={s}>
            {labels[s]}
          </option>
        ))}
      </select>
    </label>
  )
}
