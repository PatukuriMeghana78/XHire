"""
Company Extractor Module
Robust company extraction for job emails
"""

import re


class CompanyExtractor:

    def __init__(self):
        self.generic_domains = [
            'gmail', 'yahoo', 'outlook', 'hotmail', 'aol',
            'mail', 'email', 'protonmail', 'icloud'
        ]

    def extract(self, original_text: str, lowercase_text: str,
                sender_email: str = "") -> dict:

        results = []

        # 1️⃣ Offer sentence pattern
        company = self._extract_from_offer_sentence(original_text)
        if company:
            results.append({
                "company": company,
                "confidence": 95,
                "method": "offer_sentence"
            })

        # 2️⃣ Generic "at Company"
        company = self._extract_after_at(original_text)
        if company:
            results.append({
                "company": company,
                "confidence": 85,
                "method": "at_pattern"
            })

        # 3️⃣ Team mention
        company = self._extract_from_team(original_text)
        if company:
            results.append({
                "company": company,
                "confidence": 80,
                "method": "team_mention"
            })

        # 4️⃣ Email domain fallback
        company = self._extract_from_email(sender_email)
        if company:
            results.append({
                "company": company,
                "confidence": 70,
                "method": "email_domain"
            })

        if not results:
            return {
                "company": None,
                "confidence": 0,
                "method": "none"
            }

        return max(results, key=lambda x: x["confidence"])

    # ---------------------------------------------------

    def _extract_from_offer_sentence(self, text: str):

        pattern = (
            r'offer\s+you\s+the\s+position\s+of\s+.+?\s+at\s+'
            r'([A-Z][A-Za-z0-9&\.\-]{1,40})(?:[\.,\n]|$)'
        )

        match = re.search(pattern, text, re.IGNORECASE)
        if match:
            return self._clean_company(match.group(1))

        return None

    # ---------------------------------------------------

    def _extract_after_at(self, text: str):

        pattern = r'\bat\s+([A-Z][A-Za-z0-9&\.\-]{1,40})(?:[\.,\n]|$)'

        match = re.search(pattern, text)
        if match:
            return self._clean_company(match.group(1))

        return None

    # ---------------------------------------------------

    def _extract_from_team(self, text: str):

        pattern = r'([A-Z][A-Za-z0-9&\.\-]{1,40})\s+(Recruiting|Talent|Hiring|HR)\s+Team'

        match = re.search(pattern, text)
        if match:
            return self._clean_company(match.group(1))

        return None

    # ---------------------------------------------------

    def _extract_from_email(self, email: str):

        if not email:
            return None

        match = re.search(r'@([\w\.-]+)', email.lower())
        if not match:
            return None

        domain_parts = match.group(1).split('.')
        if len(domain_parts) >= 2:
            main = domain_parts[0]

            if main in self.generic_domains:
                return None

            return main.capitalize()

        return None

    # ---------------------------------------------------

    def _clean_company(self, company: str):

        company = company.strip().rstrip('.,;:!')

        company = re.sub(
            r'\s+(Inc|LLC|Ltd|Corp|Corporation|Team|Recruiting)\.?$',
            '',
            company,
            flags=re.IGNORECASE
        )

        company = ' '.join(word.capitalize() for word in company.split())

        if 2 <= len(company) <= 40:
            return company

        return None