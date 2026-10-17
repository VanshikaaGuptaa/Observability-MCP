
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
  title: string;
  summary: string;
  matched_on: string[];
  steps: string[];
  error_signature?: string;
  source?: string;
  verified?: boolean;
  match_origin?: "runbook" | "learned" | "saved";
}

export interface InvestigationResponse {
  service: string;
  duration: string;
  evidence: IncidentEvidence;
  ai_explanation: string;
  duration_ms: number;
  knowledge: KnowledgeArticle[] | null;
}

export type FailureScenario =
  | "normal"
  | "db_failure"
  | "redis_failure"
  | "payment_failure"
  | "error_spike"
  | "slow";
