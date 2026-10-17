
export interface EvidenceSignal {
  signal: string;
  source: string;
  value: string | number | boolean | null;
}

export interface IncidentImpact {
  affected_paths: string[];
  description: string;
  detected: boolean;
  http_error_rate: number;
}

export interface IncidentEvidence {
  service: string;
  status: string;
  severity: string;
  evidence: EvidenceSignal[];
  hypotheses: unknown[];
  impact: IncidentImpact;
  symptoms: unknown[];
  timeline: unknown[];
}

export interface KnowledgeArticle {
  id?: string;
  title: string;
  summary: string;
  matched_on?: string[];
  steps?: string[];
  error_signature?: string;
  source?: string;
  verified?: boolean;
  match_origin?: "runbook" | "learned" | "saved" | "static";
}

export interface InvestigationResponse {
  service: string;
  duration: string;
  evidence: IncidentEvidence;
  ai_explanation: string;
  duration_ms: number;
  knowledge: KnowledgeArticle[] | null;
  investigation_mode?: string;
  ai_skipped?: boolean;
  match_score?: number;
  decision_reason?: string;
  ai_error?: string;
}

export type GeminiKeySource = "user" | "environment" | "none";

export interface GeminiKeyStatus {
  configured: boolean;
  source: GeminiKeySource;
  env_fallback_available: boolean;
  user_key_configured: boolean;
}

export interface GeminiTestResponse extends GeminiKeyStatus {
  ok: boolean;
  message: string;
}

export type FailureScenario =
  | "normal"
  | "db_failure"
  | "redis_failure"
  | "payment_failure"
  | "error_spike"
  | "slow";
