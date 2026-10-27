import type { InputHTMLAttributes } from 'react'
import type { User } from '../api'
import { genderLabels, relationLabels, todayISO } from '../format'

export type ProfileData = Pick<
  User,
  | 'email'
  | 'firstName'
  | 'lastName'
  | 'birthDate'
  | 'gender'
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
  street: '',
  zip: '',
  city: '',
  emergencyName: '',
  emergencyPhone: '',
  emergencyRelation: '',
}

type TextField = Exclude<keyof ProfileData, 'gender' | 'emergencyRelation'>
type SelectField = 'gender' | 'emergencyRelation'

/**
 * Felder des Profils, gleich für Benutzer und Gäste. Das Profil gilt für alle Projekte,
 * der Status im Projekt wird separat gewählt (StatusSelect).
 * email "readonly": Email aus dem Google- oder Apple-Konto, "optional": Gäste ohne Konto.
 */
export default function ProfileFields({
  value,
  onChange,
  email,
}: {
  value: ProfileData
  onChange: (value: ProfileData) => void
  email: 'readonly' | 'optional'
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

  function select(field: SelectField, label: string, options: Record<string, string>) {
    return (
      <label>
        {label}
        <select required value={value[field]} onChange={(e) => update(field, e.target.value)}>
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
