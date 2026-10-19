import { useEffect, useRef, type PointerEvent } from 'react'

/** Feld für eine handschriftliche Unterschrift (Finger, Stift oder Maus). */
export default function SignaturePad({ onChange }: { onChange: (dataUrl: string | null) => void }) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const drawing = useRef(false)

  useEffect(() => {
    const canvas = canvasRef.current!
    // Auflösung an das Display anpassen, damit die Linie scharf bleibt (max. 2x, hält das PNG klein)
    const scale = Math.min(window.devicePixelRatio || 1, 2)
    canvas.width = canvas.clientWidth * scale
    canvas.height = canvas.clientHeight * scale
    const ctx = canvas.getContext('2d')!
    ctx.scale(scale, scale)
    ctx.lineWidth = 2.5
    ctx.lineCap = 'round'
    ctx.lineJoin = 'round'
    ctx.strokeStyle = '#000'
  }, [])

  function point(e: PointerEvent<HTMLCanvasElement>) {
    const rect = e.currentTarget.getBoundingClientRect()
    return [e.clientX - rect.left, e.clientY - rect.top] as const
  }

  function start(e: PointerEvent<HTMLCanvasElement>) {
    e.currentTarget.setPointerCapture(e.pointerId)
    drawing.current = true
    const ctx = e.currentTarget.getContext('2d')!
    const [x, y] = point(e)
    ctx.beginPath()
    ctx.moveTo(x, y)
    // Punkt zeichnen, auch wenn der Stift nicht bewegt wird
    ctx.lineTo(x + 0.1, y + 0.1)
    ctx.stroke()
  }

  function move(e: PointerEvent<HTMLCanvasElement>) {
    if (!drawing.current) return
    const ctx = e.currentTarget.getContext('2d')!
    const [x, y] = point(e)
    ctx.lineTo(x, y)
    ctx.stroke()
  }

  function end(e: PointerEvent<HTMLCanvasElement>) {
    if (!drawing.current) return
    drawing.current = false
    onChange(e.currentTarget.toDataURL('image/png'))
  }

  function clear() {
    const canvas = canvasRef.current!
    canvas.getContext('2d')!.clearRect(0, 0, canvas.width, canvas.height)
    onChange(null)
  }

  return (
    <div className="stack">
      <canvas
        ref={canvasRef}
        className="signature-pad"
        onPointerDown={start}
        onPointerMove={move}
        onPointerUp={end}
        onPointerCancel={end}
      />
      <div className="actions">
        <button type="button" className="link" onClick={clear}>
          Unterschrift löschen
        </button>
      </div>
    </div>
  )
}
