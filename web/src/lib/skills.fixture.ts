import type { CvDocument, Dataset } from './dataset'

/** A small dataset for the skills tests: every level, a project, a position,
 *  an expired and a current certificate, a skill with no evidence and a
 *  reference that points at nothing. The build clock is 2026-06-15. */
function doc(language: string, suffix: string): CvDocument {
  return {
    language,
    experience: [{ id: 'exp-1', employer: `Employer ${suffix}`, title: `Engineer ${suffix}` }],
    certificates: [
      {
        id: 'cert-old',
        name: `Old Cert ${suffix}`,
        issuer: `Old Body ${suffix}`,
        issued: '2022-01',
        valid_until: '2026-01',
      },
      {
        id: 'cert-new',
        name: `New Cert ${suffix}`,
        issuer: `New Body ${suffix}`,
        issued: '2025-01',
        valid_until: '2027-12',
      },
      {
        id: 'cert-forever',
        name: `Forever Cert ${suffix}`,
        issuer: `Forever Body ${suffix}`,
        issued: '2020-01',
        valid_until: null,
      },
    ],
    projects: [{ id: 'proj-1', github_repo: 'example/repo', description: `Text ${suffix}` }],
    skills: [
      {
        id: 's-k8s',
        name: `Kubernetes ${suffix}`,
        level: 'expert',
        evidence: [
          { type: 'certificate', ref: 'cert-old' },
          { type: 'certificate', ref: 'cert-new' },
          { type: 'project', ref: 'proj-1' },
        ],
      },
      {
        id: 's-go',
        name: `Go ${suffix}`,
        level: 'working',
        evidence: [{ type: 'experience', ref: 'exp-1' }],
      },
      { id: 's-rust', name: `Rust ${suffix}`, level: 'familiar', evidence: [] },
      {
        id: 's-tf',
        name: `OpenTofu ${suffix}`,
        level: 'proficient',
        evidence: [{ type: 'certificate', ref: 'cert-forever' }],
      },
      {
        id: 's-gone',
        name: `Dangling ${suffix}`,
        level: 'proficient',
        evidence: [{ type: 'project', ref: 'proj-missing' }],
      },
    ],
  }
}

export const fixtureDataset: Dataset = {
  generated_at: '2026-06-15T00:00:00Z',
  cv: { en: doc('en', 'en'), de: doc('de', 'de') },
}
