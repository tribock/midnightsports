import type { ReactNode } from 'react'
import { createPortal } from 'react-dom'

/**
 * Druckansicht (A4) mit Kopfzeile. Auf dem Bildschirm unsichtbar, beim Drucken wird nur sie ausgegeben.
 * Wird direkt in <body> gerendert, damit der Rest der Seite beim Drucken ausgeblendet werden kann.
 */
export default function PrintSheet({ title, children }: { title: string; children: ReactNode }) {
  return createPortal(
    <div className="print-sheet">
      <header className="sheet-header">
        <img src="/icon.svg" alt="" width={56} height={56} />
        <strong className="sheet-org">Midnightsports</strong>
      </header>
      <h1 className="sheet-title">{title}</h1>
      {children}
    </div>,
    document.body,
  )
}

/** Öffnet den Druckdialog. Der Titel wird als Dateiname vorgeschlagen, wenn als PDF gespeichert wird. */
export function printDocument(filename: string) {
  const previous = document.title
  document.title = filename
  window.addEventListener('afterprint', () => (document.title = previous), { once: true })
  window.print()
}
