import { useCallback, useEffect, useMemo, useState } from "react";
import type { GeminiKeyStatus, KnowledgeArticle } from "./types";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

type FilterType = "all" | "static" | "learned" | "verified" | "unverified";

const originLabels: Record<string, string> = {
  runbook: "Predefined runbook",
  static: "Predefined runbook",
  learned: "Newly learned",
  saved: "Reused from knowledge.json",
};

export function KnowledgeHubPage({
  geminiStatus,
  onOpenSettings,
}: {
  geminiStatus?: GeminiKeyStatus | null;
  onOpenSettings?: () => void;
}) {
  const [articles, setArticles] = useState<KnowledgeArticle[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [activeFilter, setActiveFilter] = useState<FilterType>("all");
  const [savingId, setSavingId] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveSuccess, setSaveSuccess] = useState<string | null>(null);
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());

  const fetchArticles = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const response = await fetch("/api/knowledge", { cache: "no-store" });
      if (!response.ok) {
        throw new Error(`Failed to load knowledge articles (${response.status}).`);
      }
      const data = await response.json();
      const list: KnowledgeArticle[] = Array.isArray(data)
        ? data
        : (data?.articles ?? []);
      setArticles(list);
    } catch (err: unknown) {
      setError(
        err instanceof Error ? err.message : "Failed to load knowledge articles.",
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchArticles();
  }, [fetchArticles]);

  const toggleExpand = (id: string) => {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const toggleVerification = async (article: KnowledgeArticle) => {
    const targetId = article.id;
    if (!targetId || article.match_origin === "static" || savingId) {
      return;
    }

    const previousVerified = Boolean(article.verified);
    const nextVerified = !previousVerified;

    setSavingId(targetId);
    setSaveError(null);
    setSaveSuccess(null);

    // Optimistic UI update
    setArticles((prev) =>
      prev.map((a) =>
        a.id === targetId ? { ...a, verified: nextVerified } : a,
      ),
    );

    try {
      const response = await fetch(
        `/api/knowledge/${encodeURIComponent(targetId)}/verification`,
        {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ verified: nextVerified }),
        },
      );

      if (!response.ok) {
        let msg = `Update failed (${response.status}).`;
        try {
          const body = (await response.json()) as { error?: string };
          if (body?.error) msg = body.error;
        } catch {
          // ignore
        }
        throw new Error(msg);
      }

      const updated = (await response.json()) as KnowledgeArticle;
      // Authoritative update from backend response
      setArticles((prev) =>
        prev.map((a) => (a.id === targetId ? { ...a, ...updated } : a)),
      );
      setSaveSuccess(
        `Article "${article.title}" is now marked as ${
          nextVerified ? "verified (eligible for router skip)" : "unverified"
        }.`,
      );
    } catch (err: unknown) {
      // Revert displayed state if write fails
      setArticles((prev) =>
        prev.map((a) =>
          a.id === targetId ? { ...a, verified: previousVerified } : a,
        ),
      );
      setSaveError(
        err instanceof Error
          ? err.message
          : "Failed to persist verification status.",
      );
      // Reload authoritative state from server
      void fetchArticles();
    } finally {
      setSavingId(null);
    }
  };

  // Counts
  const counts = useMemo(() => {
    const total = articles.length;
    let staticCount = 0;
    let learnedCount = 0;
    let verifiedCount = 0;
    let unverifiedCount = 0;

    for (const a of articles) {
      if (a.match_origin === "static") {
        staticCount++;
      } else {
        learnedCount++;
      }
      if (a.verified) {
        verifiedCount++;
      } else {
        unverifiedCount++;
      }
    }
    return {
      total,
      static: staticCount,
      learned: learnedCount,
      verified: verifiedCount,
      unverified: unverifiedCount,
    };
  }, [articles]);

  // Filtering & search
  const filteredArticles = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    return articles.filter((a) => {
      // Category filter
      if (activeFilter === "static" && a.match_origin !== "static") return false;
      if (activeFilter === "learned" && a.match_origin === "static") return false;
      if (activeFilter === "verified" && !a.verified) return false;
      if (activeFilter === "unverified" && a.verified) return false;

      // Search query filter
      if (!query) return true;

      const titleMatch = a.title?.toLowerCase().includes(query);
      const summaryMatch = a.summary?.toLowerCase().includes(query);
      const sigMatch = a.error_signature?.toLowerCase().includes(query);
      const matchTags = a.matched_on?.some((m) => m.toLowerCase().includes(query));
      const stepsMatch = a.steps?.some((s) => s.toLowerCase().includes(query));

      return titleMatch || summaryMatch || sigMatch || matchTags || stepsMatch;
    });
  }, [articles, activeFilter, searchQuery]);

  return (
    <div className="knowledge-hub-page">
      <header className="topbar">
        <div>
          <div className="eyebrow">ORGANIZATIONAL MEMORY / RUNBOOKS & INCIDENTS</div>
          <h1>Knowledge Hub</h1>
          <p className="muted">
            Inspect all static runbooks and learned incident patterns. Manage
            verification to control when investigations skip AI analysis.
          </p>
        </div>
        <div className="topbar-actions">
          {onOpenSettings && (
            <button
              type="button"
              id="knowledge-topbar-gemini"
              className={`gemini-chip ${
                !geminiStatus
                  ? "unknown"
                  : geminiStatus.configured
                    ? "on"
                    : "off"
              }`}
              onClick={onOpenSettings}
              title="Configure Gemini API key"
            >
              <span className="chip-dot" />
              {!geminiStatus
                ? "GEMINI · SETTINGS"
                : geminiStatus.configured
                  ? `GEMINI · ${geminiStatus.source === "user" ? "YOUR KEY" : "ENV KEY"}`
                  : "GEMINI · NOT CONFIGURED"}
            </button>
          )}
          <button
            type="button"
            className="button button-secondary"
            onClick={() => void fetchArticles()}
            disabled={loading}
          >
            {loading ? "Refreshing…" : "↻ Refresh Knowledge"}
          </button>
          <div className="environment-pill">
            <span className="connection-dot" /> LOCAL DEMO
          </div>
        </div>
      </header>

      {/* Summary Counts Bar */}
      <section className="metric-grid">
        <article className="metric-card">
          <div className="metric-top">
            <span className="metric-label">TOTAL ARTICLES</span>
            <span className="metric-icon purple">▤</span>
          </div>
          <div className="metric-value">{counts.total}</div>
          <div className="metric-detail">All knowledge records</div>
        </article>

        <article className="metric-card">
          <div className="metric-top">
            <span className="metric-label">STATIC RUNBOOKS</span>
            <span className="metric-icon blue">◈</span>
          </div>
          <div className="metric-value">{counts.static}</div>
          <div className="metric-detail">Built-in system runbooks</div>
        </article>

        <article className="metric-card">
          <div className="metric-top">
            <span className="metric-label">LEARNED INCIDENTS</span>
            <span className="metric-icon green">✦</span>
          </div>
          <div className="metric-value">{counts.learned}</div>
          <div className="metric-detail">Captured from past triage</div>
        </article>

        <article className="metric-card">
          <div className="metric-top">
            <span className="metric-label">ROUTER-ELIGIBLE</span>
            <span className="metric-icon green">✓</span>
          </div>
          <div className="metric-value">{counts.verified}</div>
          <div className="metric-detail">Verified incident patterns</div>
        </article>
      </section>

      {/* Search and Filters Toolbar */}
      <section className="knowledge-toolbar panel">
        <div className="toolbar-search">
          <span className="search-icon">🔍</span>
          <input
            type="text"
            className="search-input"
            placeholder="Search by title, summary, dependency, error signature, or steps…"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
          {searchQuery && (
            <button
              type="button"
              className="search-clear"
              onClick={() => setSearchQuery("")}
              title="Clear search"
            >
              ✕
            </button>
          )}
        </div>

        <div className="toolbar-filters">
          <button
            type="button"
            className={`filter-btn ${activeFilter === "all" ? "active" : ""}`}
            onClick={() => setActiveFilter("all")}
          >
            All <span className="filter-count">{counts.total}</span>
          </button>
          <button
            type="button"
            className={`filter-btn ${activeFilter === "static" ? "active" : ""}`}
            onClick={() => setActiveFilter("static")}
          >
            Static <span className="filter-count">{counts.static}</span>
          </button>
          <button
            type="button"
            className={`filter-btn ${activeFilter === "learned" ? "active" : ""}`}
            onClick={() => setActiveFilter("learned")}
          >
            Learned <span className="filter-count">{counts.learned}</span>
          </button>
          <button
            type="button"
            className={`filter-btn ${activeFilter === "verified" ? "active" : ""}`}
            onClick={() => setActiveFilter("verified")}
          >
            Verified <span className="filter-count">{counts.verified}</span>
          </button>
          <button
            type="button"
            className={`filter-btn ${activeFilter === "unverified" ? "active" : ""}`}
            onClick={() => setActiveFilter("unverified")}
          >
            Unverified <span className="filter-count">{counts.unverified}</span>
          </button>
        </div>
      </section>

      {/* Save Success / Error Feedback */}
      {saveSuccess && (
        <div className="notice" role="status">
          <span>✓ {saveSuccess}</span>
          <button
            type="button"
            className="notice-close"
            onClick={() => setSaveSuccess(null)}
          >
            ✕
          </button>
        </div>
      )}
      {saveError && (
        <div className="error-banner" role="alert">
          <span>⚠ {saveError}</span>
          <button
            type="button"
            className="notice-close"
            onClick={() => setSaveError(null)}
          >
            ✕
          </button>
        </div>
      )}
      {error && (
        <div className="error-banner" role="alert">
          <span>⚠ {error}</span>
          <button
            type="button"
            className="button button-secondary"
            onClick={() => void fetchArticles()}
            style={{ marginLeft: 12, padding: "4px 8px", fontSize: 11 }}
          >
            Try again
          </button>
        </div>
      )}

      {/* Main Articles List / Grid */}
      {loading && articles.length === 0 ? (
        <div className="panel empty-state">
          <div className="spinner" style={{ width: 24, height: 24, margin: "0 auto 12px" }} />
          <strong>Loading Knowledge Articles…</strong>
          <p>Fetching static runbooks and learned patterns from memory and storage.</p>
        </div>
      ) : filteredArticles.length === 0 ? (
        <div className="panel empty-state">
          <div className="empty-icon">▤</div>
          <strong>No matching knowledge articles</strong>
          <p>
            {searchQuery
              ? `No articles matched query "${searchQuery}". Try different keywords or reset filters.`
              : "No articles found in this category."}
          </p>
          {(searchQuery || activeFilter !== "all") && (
            <button
              type="button"
              className="button button-secondary"
              onClick={() => {
                setSearchQuery("");
                setActiveFilter("all");
              }}
              style={{ marginTop: 12 }}
            >
              Reset search & filters
            </button>
          )}
        </div>
      ) : (
        <div className="knowledge-cards-list">
          {filteredArticles.map((article, idx) => {
            const articleId = article.id || `idx-${idx}`;
            const isStatic = article.match_origin === "static";
            const isSavingThis = savingId === articleId;
            const isExpanded = expandedIds.has(articleId);

            return (
              <article
                className={`hub-card ${isStatic ? "static-card" : "learned-card"}`}
                key={articleId}
                id={`article-${articleId}`}
              >
                <div className="hub-card-top">
                  <div className="hub-card-badges">
                    <span className="book-icon">▤</span>
                    <span className={`origin-badge ${article.match_origin ?? "unknown"}`}>
                      {originLabels[article.match_origin ?? ""] ?? "Learned"}
                    </span>
                    {article.source && (
                      <span className="source-tag">Source: {article.source}</span>
                    )}
                  </div>

                  {/* Verification Controls */}
                  <div className="verification-wrapper">
                    {isStatic ? (
                      <span
                        className="verification-badge static-verified"
                        title="Static runbooks are built-in and always verified"
                      >
                        ✓ Verified (Static)
                      </span>
                    ) : (
                      <button
                        type="button"
                        className={`verification-toggle ${
                          article.verified ? "is-verified" : "is-unverified"
                        }`}
                        onClick={() => void toggleVerification(article)}
                        disabled={isSavingThis}
                        title={
                          article.verified
                            ? "Click to mark as unverified (will not skip AI)"
                            : "Click to verify (allows incident router to skip AI if matched)"
                        }
                      >
                        {isSavingThis ? (
                          <>
                            <span className="status-spinner" /> Saving…
                          </>
                        ) : article.verified ? (
                          <>✓ Verified</>
                        ) : (
                          <>○ Unverified</>
                        )}
                      </button>
                    )}
                  </div>
                </div>

                <h3 className="hub-card-title">{article.title}</h3>

                <div className="hub-card-summary markdown-content">
                  <ReactMarkdown remarkPlugins={[remarkGfm]}>
                    {article.summary}
                  </ReactMarkdown>
                </div>

                {/* Match conditions / Error signature */}
                {(article.error_signature || (article.matched_on && article.matched_on.length > 0)) && (
                  <div className="hub-signatures">
                    {article.error_signature && (
                      <div className="signature-row">
                        <span className="sig-label">Signature:</span>
                        <code className="sig-code">{article.error_signature}</code>
                      </div>
                    )}
                    {article.matched_on && article.matched_on.length > 0 && (
                      <div className="tag-list">
                        {article.matched_on.map((match, mIdx) => (
                          <span className="match-tag" key={`${match}-${mIdx}`}>
                            {match}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                )}

                {/* Collapsible Details: Steps & Metadata */}
                <div className="hub-expandable-section">
                  <button
                    type="button"
                    className="hub-expand-btn"
                    onClick={() => toggleExpand(articleId)}
                    aria-expanded={isExpanded}
                  >
                    <span>{isExpanded ? "▾ Hide resolution steps & metadata" : "▸ View resolution steps & metadata"}</span>
                    {article.steps && article.steps.length > 0 && (
                      <span className="steps-count-pill">{article.steps.length} steps</span>
                    )}
                  </button>

                  {isExpanded && (
                    <div className="hub-expanded-content">
                      {article.steps && article.steps.length > 0 && (
                        <div className="expanded-steps">
                          <strong className="steps-title">Recommended Resolution Steps:</strong>
                          <ol className="steps-list">
                            {article.steps.map((step, sIdx) => (
                              <li key={`${sIdx}-${step}`}>{step}</li>
                            ))}
                          </ol>
                        </div>
                      )}

                      <dl className="hub-metadata-grid">
                        <div>
                          <dt>ARTICLE ID</dt>
                          <dd><code>{article.id || "—"}</code></dd>
                        </div>
                        <div>
                          <dt>ORIGIN TYPE</dt>
                          <dd>{isStatic ? "Predefined static code" : "Learned AI pattern"}</dd>
                        </div>
                        <div>
                          <dt>ROUTER STATUS</dt>
                          <dd>
                            {article.verified
                              ? "Eligible to skip Gemini if matched above 90%"
                              : "Display only; Gemini analysis will run"}
                          </dd>
                        </div>
                        <div>
                          <dt>MODIFIABLE</dt>
                          <dd>{isStatic ? "No (locked in code)" : "Yes (user-controlled toggle)"}</dd>
                        </div>
                      </dl>
                    </div>
                  )}
                </div>
              </article>
            );
          })}
        </div>
      )}
    </div>
  );
}
