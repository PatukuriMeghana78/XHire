"""
Role Extractor Module
Extracts job role/title from emails
"""

import re

class RoleExtractor:
    """
    Extracts job role using pattern matching and validation.
    
    Strategies:
    1. Direct role mentions ("for the X position")
    2. Subject line parsing
    3. Keyword validation
    """
    
    def __init__(self):
        # Common job title keywords for validation
        self.title_keywords = [
            'engineer', 'developer', 'programmer', 'architect',
            'analyst', 'scientist', 'researcher',
            'manager', 'director', 'lead', 'head', 'chief',
            'designer', 'developer', 'ux', 'ui',
            'consultant', 'specialist', 'coordinator',
            'intern', 'associate', 'assistant',
            'officer', 'administrator', 'technician',
            'sales', 'marketing', 'product', 'project',
            'senior', 'junior', 'staff', 'principal'
        ]
    
    def extract(self, cleaned_text: str, subject_line: str = "") -> dict:
        """
        Extract job role with confidence score.
        
        Returns:
            {
                'role': str or None,
                'confidence': int (0-100),
                'method': str
            }
        """
        # Strategy 1: Pattern matching in email body
        role_pattern = self._extract_from_patterns(cleaned_text)
        
        # Strategy 2: Subject line
        role_subject = self._extract_from_subject(subject_line)
        
        # Prioritize pattern match (more reliable)
        if role_pattern:
            validated = self._validate_role(role_pattern)
            return {
                'role': role_pattern,
                'confidence': 95 if validated else 70,
                'method': 'pattern_matching' + ('_validated' if validated else '')
            }
        
        # Fall back to subject line
        if role_subject:
            validated = self._validate_role(role_subject)
            return {
                'role': role_subject,
                'confidence': 80 if validated else 60,
                'method': 'subject_line' + ('_validated' if validated else '')
            }
        
        # No role found
        return {
            'role': None,
            'confidence': 0,
            'method': 'none'
        }
    
    def _extract_from_patterns(self, text: str) -> str:
        """
        Extract role using common phrase patterns.
        """
        patterns = [
            # "for the Software Engineer position"
            r'for\s+the\s+([A-Z][A-Za-z0-9\s\-/]{3,50}?)\s+(?:position|role|opportunity)',
            
            # "applying to the Data Analyst position"
            r'applying\s+(?:to|for)\s+the\s+([A-Z][A-Za-z0-9\s\-/]{3,50}?)\s+(?:position|role)',
            
            # "your Software Engineer application"
            r'your\s+([A-Z][A-Za-z0-9\s\-/]{3,50}?)\s+application',
            
            # "the Product Manager role"
            r'the\s+([A-Z][A-Za-z0-9\s\-/]{3,50}?)\s+(?:role|position)',
            
            # "interest in the UX Designer position"
            r'interest\s+in\s+(?:the\s+)?([A-Z][A-Za-z0-9\s\-/]{3,50}?)\s+(?:position|role)',
            
            # "regarding the Senior Analyst position"
            r'regarding\s+(?:the|your)\s+([A-Z][A-Za-z0-9\s\-/]{3,50}?)\s+(?:position|role|application)',
            # "for the Backend Engineer position"
            r'for\s+the\s+(Backend\s+Engineer)\s+(?:position|role|opportunity)',

            # "for the Frontend Engineer position"
            r'for\s+the\s+(Frontend\s+Engineer)\s+(?:position|role|opportunity)',

            # "your Backend Engineer application"
            r'your\s+(Backend\s+Engineer)\s+application',

            # "your Frontend Engineer application"
            r'your\s+(Frontend\s+Engineer)\s+application',

            # "interest in the Backend Engineer role"
            r'interest\s+in\s+(?:the\s+)?(Backend\s+Engineer)\s+(?:position|role)',

            # "interest in the Frontend Engineer role"
            r'interest\s+in\s+(?:the\s+)?(Frontend\s+Engineer)\s+(?:position|role)'

            # "Backend Engineer at"
            r'\b([A-Za-z][A-Za-z0-9\s\-/]{3,50}?)\s+at\s+[A-Za-z][A-Za-z0-9&\.\s]{2,40}',

            # "Offer for Backend Engineer "
            r'for\s+([A-Za-z][A-Za-z0-9\s\-/]{3,50}?)\s+at\s+[A-Za-z][A-Za-z0-9&\.\s]{2,40}', 
            # "position of Backend Engineer "
            r'position\s+of\s+([A-Za-z][A-Za-z0-9\s\-/]{3,50}?)\s+at\s+[A-Za-z][A-Za-z0-9&\.\s]{2,40}'  

        ]
        
        for pattern in patterns:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                role = match.group(1).strip()
                role = ' '.join(word.capitalize() for word in role.split())
                
                # Clean up common noise words at the end
                role = re.sub(r'\s+(?:at|with|in)$', '', role)
                
                # Validate: reasonable length
                if 3 <= len(role) <= 50:
                    return role
        
        return None
    
    def _extract_from_subject(self, subject: str) -> str:
        """
        Extract role from subject line.
        
        Common patterns:
        - "Software Engineer - Google"
        - "Re: Product Manager Position"
        - "Application: Data Analyst"
        """
        if not subject:
            return None
        
        patterns = [
            # "Software Engineer - Company"
            r'^([[A-Za-z]][A-Za-z0-9\s\-/]{3,50}?)\s*[-–—]\s*[A-Z]',
            
            # "Position: Software Engineer"
            r'(?:Position|Role|Job):\s*([[A-Za-z]][A-Za-z0-9\s\-/]{3,50})(?:\s|$)',
            
            # "Application for Software Engineer"
            r'(?:Application|Applying)\s+for\s+([[A-Za-z]][A-Za-z0-9\s\-/]{3,50})(?:\s|$)',
        ]
        
        for pattern in patterns:
            match = re.search(pattern, subject, re.IGNORECASE)
            if match:
                role = match.group(1).strip()
                
                # Clean up
                role = re.sub(r'\s+(?:at|with|position|role).*$', '', role, flags=re.IGNORECASE)
                
                if 3 <= len(role) <= 50:
                    return role
        
        # Fallback: Check if subject contains job keywords
        for keyword in self.title_keywords:
            if keyword in subject.lower():
                # Try to extract a phrase containing the keyword
                pattern = r'([A-Za-z]+\s+)?' + keyword + r'(\s+[A-Za-z]+)?'
                match = re.search(pattern, subject, re.IGNORECASE)
                if match:
                    role = match.group(0).strip()
                    if len(role) >= 3:
                        # Capitalize properly
                        return ' '.join(word.capitalize() for word in role.split())
        
        return None
    
    def _validate_role(self, role: str) -> bool:
        """
        Validate that extracted text looks like a job title.
        
        Checks if role contains at least one job-related keyword.
        """
        role_lower = role.lower()
        return any(keyword in role_lower for keyword in self.title_keywords)