import type { UserStatus } from '../api'
import { statusLabels } from '../format'

/**
 * Status im Projekt (Herkunft). Mitarbeiter vergeben nur Projekt-Admins (canSetStaff),
 * alle anderen wählen zwischen Schüler/in aus der Region und Gast extern.
 */
export default function StatusSelect({
  value,
  onChange,
  region,
  canSetStaff = false,
  label = 'Status',
}: {
  value: UserStatus | ''
  onChange: (value: UserStatus) => void
  region: string
  canSetStaff?: boolean
  label?: string
}) {
  const labels = statusLabels(region)
  const options = (Object.keys(labels) as UserStatus[]).filter((s) => canSetStaff || s !== 'mitarbeiter')
  return (
    <label>
      {label}
      <select required value={value} onChange={(e) => onChange(e.target.value as UserStatus)}>
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
