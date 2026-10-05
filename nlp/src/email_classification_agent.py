"""
Email Classification Agent (AI Suggestion Layer)
"""

from .preprocessor import Preprocessor
from .company_extractor import CompanyExtractor
from .role_extractor import RoleExtractor


class EmailClassificationAgent:

    def __init__(self):
        self.preprocessor = Preprocessor()
        self.company_extractor = CompanyExtractor()
        self.role_extractor = RoleExtractor()

        self.keyword_weights = {
            "Interview": {
                "interview": 3,
                "schedule": 2,
                "technical round": 3,
                "hr round": 2,
                "meeting link": 2,
                "zoom": 1
            },
            "Rejected": {
                "regret": 3,
                "unfortunately": 3,
                "not selected": 4,
                "not moving forward": 4,
                "unsuccessful": 3
            },
            "Offer": {
                "offer": 4,
                "congratulations": 3,
                "pleased to offer": 4,
                "compensation": 3,
                "salary": 2,
                "joining date": 3
            },
            "Applied": {
                "application received": 4,
                "thank you for applying": 3,
                "we have received": 3,
                "under review": 2
            }
        }

    def classify(self, email_text: str) -> dict:

        # 1️⃣ Preprocess
        processed = self.preprocessor.clean(email_text)
        subject = self.preprocessor.extract_subject(email_text)
        sender_email = self.preprocessor.extract_sender_email(email_text)

        cleaned_text = processed["cleaned"]
        lowercase_text = processed["lowercase"]

        # ✅ 2️⃣ USE ORIGINAL TEXT FOR COMPANY EXTRACTION
        company_result = self.company_extractor.extract(
            email_text,              # 🔥 THIS IS THE FIX
            email_text.lower(),
            sender_email
        )

        # 3️⃣ Role extraction
        role_result = self.role_extractor.extract(
            cleaned_text,
            subject
        )

        # 4️⃣ Status scoring
        scores = {status: 0 for status in self.keyword_weights}
        matched_keywords = {status: [] for status in self.keyword_weights}

        for status, keywords in self.keyword_weights.items():
            for keyword, weight in keywords.items():
                if keyword in lowercase_text:
                    scores[status] += weight
                    matched_keywords[status].append(keyword)

        best_status = max(scores, key=scores.get)
        best_score = scores[best_status]
        total_score = sum(scores.values())

        if total_score == 0:
            best_status = "Applied"
            confidence = 0.40
            reason = "No strong keywords detected. Defaulted to Applied."
        else:
            confidence = round(best_score / total_score, 2)
            reason = (
                f"Detected keywords: {', '.join(matched_keywords[best_status])}. "
                f"{best_status} score highest among categories."
            )

        return {
            "status": best_status,
            "company": company_result.get("company"),
            "role": role_result.get("role"),
            "confidence": confidence,
            "reason": reason
        }