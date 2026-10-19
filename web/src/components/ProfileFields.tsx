import type { InputHTMLAttributes } from 'react'
import type { User } from '../api'
import { genderLabels, relationLabels, statusLabels, todayISO } from '../format'

export type ProfileData = Pick<
  User,
  | 'email'
  | 'firstName'
  | 'lastName'
  | 'birthDate'
  | 'gender'
  | 'status'
  | 'street'
  | 'zip'
  | 'city'
  | 'emergencyName'
  | 'emergencyPhone'
  | 'emergencyRelation'
>

export const emptyProfile: ProfileData = {
  email: '',
  firstName: '',
  lastName: '',
  birthDate: '',
  gender: '',
  status: '',
  street: '',
  zip: '',
  city: '',
  emergencyName: '',
  emergencyPhone: '',
  emergencyRelation: '',
}

type TextField = Exclude<keyof ProfileData, 'gender' | 'status' | 'emergencyRelation'>
type SelectField = 'gender' | 'status' | 'emergencyRelation'

/** Status, die alle selbst wählen können. Mitarbeiter vergeben nur Admins. */
const { mitarbeiter: _, ...selfStatusLabels } = statusLabels

/**
 * Felder des Profils, gleich für Benutzer und Gäste.
 * email "readonly": Email aus dem Google-Konto, "optional": Gäste ohne Konto.
 * canSetStaffStatus: Status Mitarbeiter wählbar (nur Admins). Ohne wird ein bereits vergebener
 * Status Mitarbeiter nur angezeigt.
 */
export default function ProfileFields({
  value,
  onChange,
  email,
  canSetStaffStatus,
}: {
  value: ProfileData
  onChange: (value: ProfileData) => void
  email: 'readonly' | 'optional'
  canSetStaffStatus: boolean
}) {
  function update(field: keyof ProfileData, v: string) {
    onChange({ ...value, [field]: v })
  }

  function input(field: TextField, label: string, props: InputHTMLAttributes<HTMLInputElement> = {}) {
    return (
      <label>
        {label}
        <input required value={value[field]} onChange={(e) => update(field, e.target.value)} {...props} />
      </label>
    )
  }

  function select(field: SelectField, label: string, options: Record<string, string>, disabled = false) {
    return (
      <label>
        {label}
        <select required disabled={disabled} value={value[field]} onChange={(e) => update(field, e.target.value)}>
          <option value="" disabled>
            Bitte wählen
          </option>
          {Object.entries(options).map(([v, text]) => (
            <option key={v} value={v}>
              {text}
            </option>
          ))}
        </select>
      </label>
    )
  }

  return (
    <>
      <fieldset className="card stack">
        <legend>Person</legend>
        {email === 'readonly' ? (
          <label>
            Email
            <input value={value.email} disabled />
          </label>
        ) : (
          input('email', 'Email (freiwillig)', { type: 'email', required: false, autoComplete: 'off' })
        )}
        <div className="row">
          {input('firstName', 'Vorname', { autoComplete: 'given-name' })}
          {input('lastName', 'Name', { autoComplete: 'family-name' })}
        </div>
        <div className="row">
          {input('birthDate', 'Geburtsdatum', { type: 'date', max: todayISO(), autoComplete: 'bday' })}
          {select('gender', 'Geschlecht', genderLabels)}
        </div>
        {canSetStaffStatus
          ? select('status', 'Status', statusLabels)
          : value.status === 'mitarbeiter'
            ? select('status', 'Status (vergeben durch Admin)', statusLabels, true)
            : select('status', 'Status', selfStatusLabels)}
      </fieldset>

      <fieldset className="card stack">
        <legend>Adresse</legend>
        {input('street', 'Strasse und Nr. (freiwillig)', { required: false, autoComplete: 'street-address' })}
        <div className="row">
          {input('zip', 'PLZ', { inputMode: 'numeric', autoComplete: 'postal-code', className: 'zip' })}
          {input('city', 'Ort', { autoComplete: 'address-level2' })}
        </div>
      </fieldset>

      <fieldset className="card stack">
        <legend>Notfallkontakt</legend>
        {input('emergencyName', 'Name')}
        <div className="row">
          {input('emergencyPhone', 'Telefon', { type: 'tel' })}
          {select('emergencyRelation', 'Beziehung', relationLabels)}
        </div>
      </fieldset>
    </>
  )
}
