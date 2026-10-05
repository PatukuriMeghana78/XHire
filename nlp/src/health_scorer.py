"""
Health Scorer Module
Calculates a 0-100 health score for each job application.

Signals (total = 100 pts):
  - Status score:        30 pts  (what stage are you at?)
  - Email activity:      25 pts  (has a recruiter replied recently?)
  - Recency:             20 pts  (how long ago did you apply?)
  - Follow-up:           15 pts  (did you follow up?)
  - Email sentiment:     10 pts  (is the tone positive?)

Place this file at:  backend/nlp/src/health_scorer.py
"""

from datetime import datetime, timezone
from .confidence_scorer import ConfidenceScorer  # reuse existing module


class HealthScorer:
    """
    Calculates an AI-powered health score for a job application.
    Extends ConfidenceScorer logic with time-decay and NLP sentiment.
    """

    # ── Signal weights (must sum to 100) ──────────────────────────────────────
    WEIGHTS = {
    "status":         35,   # Primary driver — what stage are you at?
    "email_activity": 30,   # Primary driver — is there recent recruiter contact?
    "recency":        10,   # Minor — time passing alone won't kill a score
    "follow_up":      15,   # Moderate — did you take action?
    "sentiment":      10,   # Minor — tone of emails
}

    # ── Status tier mapping ────────────────────────────────────────────────────
    STATUS_SCORES = {
        "Offer":       100,
        "Interview":    85,
        "Applied":      50,
        "Ghosted":      20,
        "Rejected":      0,
    }

    # ── Positive / negative keywords for lightweight sentiment ─────────────────
    POSITIVE_KEYWORDS = [
        "pleased", "excited", "congratulations", "pleased to inform",
        "next steps", "interview", "move forward", "shortlisted",
        "impressed", "delighted", "looking forward",
    ]
    NEGATIVE_KEYWORDS = [
        "unfortunately", "regret", "not moving forward", "other candidates",
        "position has been filled", "not a fit", "unable to proceed",
        "not successful", "rejected", "no longer considering",
    ]

    def calculate(
        self,
        status: str,
        applied_date: str,
        last_updated: str,
        follow_up_sent: bool,
        email_summary: str = "",
    ) -> dict:
        """
        Main entry point. Returns full health score result dict.

        Args:
            status:          Application status string (e.g. "Interview")
            applied_date:    ISO 8601 date string of when you applied
            last_updated:    ISO 8601 date string of last status change or email
            follow_up_sent:  Whether a follow-up email was sent
            email_summary:   Optional recent email text for sentiment analysis

        Returns:
            {
                "score": int (0-100),
                "color": str ("green" | "yellow" | "red"),
                "factors": {
                    "status_score": int,
                    "recency_score": int,
                    "email_activity_score": int,
                    "follow_up_score": int,
                    "sentiment_score": int,
                },
                "reason": str
            }
        """
        now = datetime.now(timezone.utc)

        # Parse dates safely
        applied_dt = self._parse_date(applied_date)
        updated_dt = self._parse_date(last_updated)

        days_since_applied = (now - applied_dt).days if applied_dt else 30
        days_since_update  = (now - updated_dt).days  if updated_dt else 30

        # ── Calculate each signal ──────────────────────────────────────────────
        status_score         = self._score_status(status)
        recency_score        = self._score_recency(days_since_applied)
        email_activity_score = self._score_email_activity(days_since_update, status)
        follow_up_score      = self._score_follow_up(follow_up_sent, days_since_applied)
        sentiment_score      = self._score_sentiment(email_summary)

        # ── Weighted total ─────────────────────────────────────────────────────
        raw_score = (
            status_score         * self.WEIGHTS["status"]         / 100 +
            email_activity_score * self.WEIGHTS["email_activity"] / 100 +
            recency_score        * self.WEIGHTS["recency"]        / 100 +
            follow_up_score      * self.WEIGHTS["follow_up"]      / 100 +
            sentiment_score      * self.WEIGHTS["sentiment"]      / 100
        )

        final_score = max(0, min(100, round(raw_score)))
        color       = self._score_to_color(final_score)
        reason      = self._build_reason(
            status, days_since_applied, days_since_update,
            follow_up_sent, sentiment_score, final_score
        )

        return {
            "score": final_score,
            "color": color,
            "factors": {
                "status_score":         status_score,
                "recency_score":        recency_score,
                "email_activity_score": email_activity_score,
                "follow_up_score":      follow_up_score,
                "sentiment_score":      sentiment_score,
            },
            "reason": reason,
        }

    # ── Private scoring helpers ────────────────────────────────────────────────

    def _score_status(self, status: str) -> int:
        """Map application status to a 0-100 score."""
        return self.STATUS_SCORES.get(status, 40)  # unknown status → neutral 40

    def _score_recency(self, days_since_applied: int) -> int:
        """
        Gentle recency decay — time alone won't sink a healthy application.

        0–14 days  → 100  (fresh)
        15–30 days →  85  (still recent)
        31–60 days →  65  (getting older but not penalised heavily)
        61–90 days →  40  (notable staleness)
        90+ days   →  20  (old, but status/email can still keep overall score high)
        """
        if days_since_applied <= 14:
            return 100
        elif days_since_applied <= 30:
            return 85
        elif days_since_applied <= 60:
            return 65
        elif days_since_applied <= 90:
            return 40
        return 20   

    def _score_email_activity(self, days_since_update: int, status: str) -> int:
        """
        Recent email/status update = good signal.
        Rejected status overrides to 0 regardless of recency.
        """
        if status == "Rejected":
            return 0

        if days_since_update <= 3:
            return 100
        elif days_since_update <= 7:
            return 80
        elif days_since_update <= 14:
            return 60
        elif days_since_update <= 30:
            return 30
        return 0

    def _score_follow_up(self, follow_up_sent: bool, days_since_applied: int) -> int:
        """
        Follow-up sent = full 100.
        No follow-up after 7+ days = missed opportunity (penalty based on wait).
        """
        if follow_up_sent:
            return 100
        # No follow-up: penalise based on how long it's been
        if days_since_applied <= 6:
            return 80   # too early to follow up — no penalty
        elif days_since_applied <= 14:
            return 40   # should have followed up
        return 10       # definitely overdue

    def _score_sentiment(self, email_text: str) -> int:
        """
        Lightweight keyword-based sentiment scoring.
        Returns 0–100. No email text → neutral 50.
        """
        if not email_text or not email_text.strip():
            return 50   # neutral: no email data to judge

        text_lower = email_text.lower()

        positive_hits = sum(1 for kw in self.POSITIVE_KEYWORDS if kw in text_lower)
        negative_hits = sum(1 for kw in self.NEGATIVE_KEYWORDS if kw in text_lower)

        if negative_hits > 0 and positive_hits == 0:
            return 0
        elif negative_hits > 0:
            return 30   # mixed signals
        elif positive_hits >= 2:
            return 100
        elif positive_hits == 1:
            return 75
        return 50       # neutral (no keywords matched)

    def _score_to_color(self, score: int) -> str:
        if score >= 70:
            return "green"
        elif score >= 40:
            return "yellow"
        return "red"

    def _build_reason(
        self,
        status: str,
        days_since_applied: int,
        days_since_update: int,
        follow_up_sent: bool,
        sentiment_score: int,
        final_score: int,
    ) -> str:
        """Build a human-readable explanation of the score."""
        parts = []

        parts.append(f"Status is '{status}'.")

        if days_since_applied <= 7:
            parts.append("Applied recently — strong recency signal.")
        elif days_since_applied > 30:
            parts.append(f"Applied {days_since_applied} days ago — recency penalty applied.")

        if days_since_update <= 7:
            parts.append("Recent email activity detected.")
        elif days_since_update > 30:
            parts.append("No email activity in 30+ days.")

        if follow_up_sent:
            parts.append("Follow-up email was sent — good initiative.")
        elif days_since_applied > 7:
            parts.append("No follow-up sent yet — consider reaching out.")

        if sentiment_score == 0:
            parts.append("Email tone appears negative.")
        elif sentiment_score >= 75:
            parts.append("Email tone is positive.")

        parts.append(f"Overall health score: {final_score}/100.")
        return " ".join(parts)

    def _parse_date(self, date_str: str):
        """Safely parse ISO 8601 date string to aware datetime."""
        if not date_str:
            return None
        try:
            dt = datetime.fromisoformat(date_str.replace("Z", "+00:00"))
            if dt.tzinfo is None:
                dt = dt.replace(tzinfo=timezone.utc)
            return dt
        except (ValueError, AttributeError):
            return None