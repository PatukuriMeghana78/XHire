from fastapi import FastAPI
from pydantic import BaseModel
from typing import List, Optional
from datetime import datetime
from .health_scorer import HealthScorer
from .parser import JobEmailParser
from .email_classification_agent import EmailClassificationAgent
from .preprocessor import Preprocessor

app = FastAPI(title="Job Email NLP API", version="1.0.0")

# Initialize parsers
email_parser = JobEmailParser()
classification_agent = EmailClassificationAgent()
health_scorer = HealthScorer()
preprocessor = Preprocessor()

# ============================================================================
# REQUEST SCHEMAS
# ============================================================================

class AnalyzeRequest(BaseModel):
    text: str
    verbose: Optional[bool] = False

class BatchAnalyzeRequest(BaseModel):
    emails: List[str]
    verbose: Optional[bool] = False

class HealthScoreRequest(BaseModel):
    application_id: int
    company: str
    role: str
    status: str
    applied_date: str
    last_updated: str
    follow_up_sent: bool
    email_summary: Optional[str] = ""

class SkillMatchRequest(BaseModel):
    required_skills: List[str] = []
    candidate_skills: List[str] = []

# ============================================================================
# RESPONSE SCHEMAS
# ============================================================================

class AnalyzeResponse(BaseModel):
    company: Optional[str]
    role: Optional[str]
    status: str
    confidence: int

class AgentResponse(BaseModel):
    company: Optional[str]
    role: Optional[str]
    status: str
    confidence: float
    reason: str

#  Notification schemas
class NotificationSuggestion(BaseModel):
    message: str
    action: str
    priority: str
    suggested_status: str
    created_at: str

class AnalyzeWithNotificationResponse(BaseModel):
    company: Optional[str]
    role: Optional[str]
    status: str
    confidence: int
    suggestion: NotificationSuggestion

class AgentResponseWithNotification(BaseModel):
    company: Optional[str]
    role: Optional[str]
    status: str
    confidence: float
    reason: str
    suggestion: NotificationSuggestion

class HealthScoreFactors(BaseModel):
    status_score: int
    recency_score: int
    email_activity_score: int
    follow_up_score: int
    sentiment_score: int

class HealthScoreResponse(BaseModel):
    score: int
    color: str
    factors: HealthScoreFactors
    reason: str

# ============================================================================
# HELPER FUNCTIONS
# ============================================================================

def generate_notification_message(status: str, company: Optional[str], role: Optional[str]) -> str:
    """Generate user-friendly notification message with emojis"""
    company_text = company or "a company"
    role_text = role or "a position"
    
    templates = {
        'Interview': f"🎯 Interview invitation from {company_text} for {role_text}",
        'Offer': f"🎉 Job offer from {company_text} for {role_text}",
        'Rejected': f"❌ Application to {company_text} for {role_text} was not successful",
        'Applied': f"✅ Application confirmation from {company_text} for {role_text}"
    }
    
    return templates.get(status, f"Update from {company_text}")

def get_notification_priority(status: str) -> str:
    """Determine notification priority based on status"""
    if status in ['Interview', 'Offer']:
        return 'high'
    elif status == 'Rejected':
        return 'medium'
    else:
        return 'low'

# ENDPOINTS

@app.get("/")
def health():
    return {"status": "NLP service running"}


@app.post("/analyze", response_model=AnalyzeResponse)
def analyze(req: AnalyzeRequest):
    """Basic email analysis (original endpoint)"""
    result = email_parser.parse(req.text, verbose=req.verbose)
    
    return {
        "company": result.get("company"),
        "role": result.get("role"),
        "status": result.get("status"),
        "confidence": result.get("confidence"),
    }

@app.post("/analyze/batch")
def analyze_batch(req: BatchAnalyzeRequest):
    """Batch email analysis (original endpoint)"""
    results = email_parser.parse_batch(req.emails, verbose=req.verbose)
    return {"results": results}

@app.post("/classify", response_model=AgentResponse)
def classify_email(req: AnalyzeRequest):
    """
    AI Agent Classification with reasoning
    """
    result = classification_agent.classify(req.text)
    
    return {
        "company": result.get("company"),
        "role": result.get("role"),
        "status": result.get("status"),
        "confidence": result.get("confidence"),
        "reason": result.get("reason"),
    }

@app.post("/health-score", response_model=HealthScoreResponse)
def calculate_health_score(req: HealthScoreRequest):
    """
    Calculate AI-powered health score (0-100) for a job application
    (original endpoint)
    """
    result = health_scorer.calculate(
        status=req.status,
        applied_date=req.applied_date,
        last_updated=req.last_updated,
        follow_up_sent=req.follow_up_sent,
        email_summary=req.email_summary or "",
    )
    return result

@app.post("/skill-match")
def skill_match(req: SkillMatchRequest):
    """
    Score the overlap between a job's required skills and a candidate's skills.
    Reuses the existing Preprocessor for case/whitespace normalization so the
    result matches the Go MatchService fallback contract (camelCase keys).
    """
    def norm(skill: str) -> str:
        return preprocessor.clean(skill)["lowercase"].strip()

    required = {}
    for skill in req.required_skills:
        key = norm(skill)
        if key:
            required[key] = skill
    candidate = {}
    for skill in req.candidate_skills:
        key = norm(skill)
        if key:
            candidate[key] = skill

    matched = [orig for key, orig in required.items() if key in candidate]
    missing = [orig for key, orig in required.items() if key not in candidate]
    extra = [orig for key, orig in candidate.items() if key not in required]
    score = int(len(matched) / len(required) * 100) if required else 0

    return {
        "matchScore": score,
        "matchedSkills": matched,
        "missingSkills": missing,
        "extraSkills": extra,
    }

# NOTIFICATION-ENHANCED ENDPOINTS

@app.post("/analyze/with-notification", response_model=AnalyzeWithNotificationResponse)
def analyze_with_notification(req: AnalyzeRequest):
    """
    Email analysis WITH notification metadata
    Use this endpoint for the notification system
    """
    # Use existing parser
    result = email_parser.parse(req.text, verbose=req.verbose)
    
    company = result.get("company")
    role = result.get("role")
    status = result.get("status")
    confidence = result.get("confidence")
    
    # Generate notification
    message = generate_notification_message(status, company, role)
    priority = get_notification_priority(status)
    
    return {
        "company": company,
        "role": role,
        "status": status,
        "confidence": confidence,
        "suggestion": {
            "message": message,
            "action": "update_status",
            "priority": priority,
            "suggested_status": status,
            "created_at": datetime.utcnow().isoformat() + "Z"
        }
    }

@app.post("/classify/with-notification", response_model=AgentResponseWithNotification)
def classify_with_notification(req: AnalyzeRequest):
    """
    AI Agent Classification WITH notification metadata
    Includes reasoning + notification data
    """
    # Use existing classification agent
    result = classification_agent.classify(req.text)
    
    company = result.get("company")
    role = result.get("role")
    status = result.get("status")
    confidence = result.get("confidence")
    reason = result.get("reason")
    
    # Generate notification
    message = generate_notification_message(status, company, role)
    priority = get_notification_priority(status)
    
    return {
        "company": company,
        "role": role,
        "status": status,
        "confidence": confidence,
        "reason": reason,
        "suggestion": {
            "message": message,
            "action": "update_status",
            "priority": priority,
            "suggested_status": status,
            "created_at": datetime.utcnow().isoformat() + "Z"
        }
    }