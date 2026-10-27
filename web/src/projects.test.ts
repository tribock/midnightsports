import { describe, expect, it } from 'vitest'
import { statusLabels } from './format'
import { slugify } from './pages/AdminProjectsPage'
import { projectLogo } from './project'

describe('Projekte', () => {
  it('Kurzname aus dem Namen', () => {
    expect(slugify('Pleno Frauenfeld')).toBe('pleno-frauenfeld')
    expect(slugify('Jugendtreff Münchwilen')).toBe('jugendtreff-muenchwilen')
    expect(slugify('  Midnightsports (Aadorf)! ')).toBe('midnightsports-aadorf')
    expect(slugify('Café Zürich')).toBe('cafe-zuerich')
  })

  it('Status mit der Region des Projekts', () => {
    expect(statusLabels('Aadorf').schueler).toBe('Schüler-/in aus Aadorf')
    expect(statusLabels('').schueler).toBe('Schüler-/in')
    expect(statusLabels('Frauenfeld').gast_extern).toBe('Gast extern')
  })

  it('Logo: hochgeladen oder Standard', () => {
    expect(projectLogo({ slug: 'pleno', hasLogo: true })).toBe('/api/public/projects/pleno/logo')
    expect(projectLogo({ slug: 'aadorf', hasLogo: false })).toBe('/logo.svg')
    expect(projectLogo(null, 'white')).toBe('/logo-white.svg')
  })
})
