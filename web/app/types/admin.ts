/**
 * Wire contract for the administrative surface: the principal directory
 * (`/api/people`) and the key inventory (`/api/keys`).
 *
 * Kept in its own file rather than appended to `~/types/api`: these endpoints
 * are gated by the `admin` scope and their models live in
 * `internal/models/people.go` and `internal/models/api_key.go`. Every tag below
 * is transcribed from the Go struct — the two must not drift.
 *
 * The vocabulary tuples are declared before the types (and the label maps
 * after), matching the pattern `~/types/api` uses for statuses, so the select
 * menus cannot fall behind the Go constants.
 */

/** Mirrors models.PersonKind — enforced by the people_kind_check constraint. */
export const PERSON_KINDS = ['human', 'service'] as const
export type PersonKind = (typeof PERSON_KINDS)[number]

export const PERSON_KIND_LABELS: Record<PersonKind, string> = {
  human: 'Human',
  service: 'Service'
}

/**
 * Mirrors models.AllScopes. `*` (models.ScopeAll) is not offered for new keys:
 * it is a stored value an operator may encounter, not something to mint from a
 * form where three explicit checkboxes say the same thing more legibly.
 */
export const KEY_SCOPES = ['read', 'write', 'admin'] as const
export type KeyScope = (typeof KEY_SCOPES)[number]

export const KEY_SCOPE_LABELS: Record<KeyScope, string> = {
  read: 'Read',
  write: 'Write',
  admin: 'Admin'
}

/**
 * models.Person. `email` carries `json:"email,omitempty"`, so it is absent
 * rather than empty on a person created without one.
 */
export interface Person {
  id: string
  kind: PersonKind
  display_name: string
  email?: string
  created_at: string
}

/** models.CreatePersonRequest. `kind` defaults to human server-side. */
export interface CreatePersonInput {
  kind: PersonKind
  display_name: string
  email?: string
}

/**
 * models.APIKey.
 *
 * `key_hash` is tagged `json:"-"` server-side and is deliberately absent here:
 * a type that could hold a digest is a type that eventually renders one.
 *
 * `scopes` is nullable because models.NormalizeScopes returns nil for an empty
 * request, which the key validator reads as "unrestricted" (models.Allows).
 */
export interface APIKey {
  id: string
  key_prefix: string
  principal_id: string
  project_id?: string
  /**
   * The agent this credential is bound to, or absent when it is bound to none.
   *
   * Absent is the common case rather than the exception: every key issued
   * before migration 017 has no agent binding, as does the legacy shared-secret
   * path. When it *is* present, the MCP surface treats it as authoritative and
   * rejects an `agent_id` in the connection URL that names a different agent.
   */
  agent_id?: string
  scopes: KeyScope[] | null
  created_at: string
  last_used_at?: string
  revoked_at?: string
  principal_type: PersonKind
  /** Filled in by the list query's join with people; absent on the mint response. */
  principal_name?: string
}

/**
 * models.CreateAPIKeyRequest. `principal_type` is never accepted from a client.
 *
 * `agent_id` binds the key to one agent; the server then takes the key's project
 * from that agent rather than from the body, so the two cannot disagree.
 */
export interface CreateKeyInput {
  principal_id: string
  project_id?: string
  agent_id?: string
  scopes: KeyScope[]
}

/**
 * models.IssuedAPIKey — the only response in the API that carries a plaintext
 * secret, and it does so exactly once. The server does not keep it, so this
 * value is the only place it will ever exist; the UI holds it in a single ref
 * that is cleared when the operator dismisses it.
 */
export interface IssuedAPIKey extends APIKey {
  secret: string
}

/** Envelope used by GET /api/people and GET /api/keys. */
export interface PeopleResponse { people: Person[] }
export interface KeysResponse { keys: APIKey[] }