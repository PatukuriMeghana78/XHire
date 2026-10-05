"""
Preprocessor Module
Cleans and prepares email text for extraction
"""

import re

class Preprocessor:
    """
    Handles text cleaning and normalization.
    
    Purpose: Prepare raw email text for pattern matching and extraction
    """
    
    def __init__(self):
        pass
    
    def clean(self, email_text: str) -> dict:
        """
        Clean and prepare email text.
        
        Returns:
            dict with 'original', 'cleaned', and 'lowercase' versions
        """
        # Remove HTML tags
        no_html = re.sub(r'<[^>]+>', '', email_text)
        
        # Remove extra whitespace while preserving structure
        cleaned = re.sub(r'\s+', ' ', no_html)
        cleaned = cleaned.strip()
        
        # Create lowercase version for case-insensitive matching
        lowercase = cleaned.lower()
        
        return {
            'original': email_text,
            'cleaned': cleaned,
            'lowercase': lowercase
        }
    
    def extract_subject(self, email_text: str) -> str:
        """
        Extract subject line from email.
        
        Common patterns:
        - "Subject: ..."
        - "Re: ..."
        - "subject: ..."
        """
        patterns = [
            r'(?:subject|Subject|SUBJECT):\s*(.+?)(?:\n|$)',
            r'(?:re|Re|RE):\s*(.+?)(?:\n|$)'
        ]
        
        for pattern in patterns:
            match = re.search(pattern, email_text)
            if match:
                return match.group(1).strip()
        
        return ""
    
    def extract_sender_email(self, email_text: str) -> str:
        """
        Extract sender's email address.
        
        Pattern: anything@domain.com
        """
        pattern = r'[\w\.-]+@[\w\.-]+\.\w+'
        match = re.search(pattern, email_text)
        
        return match.group(0) if match else ""