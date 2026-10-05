"""
Main Parser Module
Orchestrates all extraction modules to parse job emails
"""

from .preprocessor import Preprocessor
from .company_extractor import CompanyExtractor
from .role_extractor import RoleExtractor
from .status_classifier import StatusClassifier
from .confidence_scorer import ConfidenceScorer


class JobEmailParser:
    """
    Main orchestrator for job email parsing.
    
    Coordinates all modules and produces final structured output.
    """
    
    def __init__(self):
        self.preprocessor = Preprocessor()
        self.company_extractor = CompanyExtractor()
        self.role_extractor = RoleExtractor()
        self.status_classifier = StatusClassifier()
        self.confidence_scorer = ConfidenceScorer()
    
    def parse(self, email_text: str, verbose: bool = False) -> dict:
        """
        Parse a job application email into structured data.
        
        Args:
            email_text: Raw email text
            verbose: If True, include detailed extraction information
        
        Returns:
            {
                'company': str or None,
                'role': str or None,
                'status': str ('Applied', 'Interview', 'Rejected'),
                'confidence': int (0-100),
                'details': dict (if verbose=True)
            }
        """
        # Step 1: Preprocess
        processed = self.preprocessor.clean(email_text)
        subject = self.preprocessor.extract_subject(email_text)
        sender_email = self.preprocessor.extract_sender_email(email_text)
        
        # Step 2: Extract company
        company_result = self.company_extractor.extract(
            processed['cleaned'],
            processed['lowercase'],
            sender_email
        )
        
        # Step 3: Extract role
        role_result = self.role_extractor.extract(
            processed['cleaned'],
            subject
        )
        
        # Step 4: Classify status
        status_result = self.status_classifier.classify(
            processed['lowercase']
        )
        
        # Step 5: Calculate overall confidence
        confidence_result = self.confidence_scorer.calculate_overall_confidence(
            company_result['confidence'],
            role_result['confidence'],
            status_result['confidence']
        )
        
        # Step 6: Build result
        result = {
            'company': company_result['company'],
            'role': role_result['role'],
            'status': status_result['status'],
            'confidence': confidence_result['overall']
        }
        
        # Add detailed information if requested
        if verbose:
            result['details'] = {
                'company_details': company_result,
                'role_details': role_result,
                'status_details': status_result,
                'confidence_breakdown': confidence_result,
                'explanation': self.confidence_scorer.get_confidence_explanation(
                    company_result, role_result, status_result, 
                    confidence_result['overall']
                )
            }
        
        return result
    
    def parse_batch(self, emails: list, verbose: bool = False) -> list:
        """
        Parse multiple emails.
        
        Args:
            emails: List of email text strings
            verbose: Include detailed information
        
        Returns:
            List of parsed results
        """
        return [self.parse(email, verbose) for email in emails]


# Convenience function for simple usage
def parse_job_email(email_text: str, verbose: bool = False) -> dict:
    """
    Quick function to parse a single job email.
    
    Usage:
        result = parse_job_email(email_content)
        print(result['company'], result['role'], result['status'])
    """
    parser = JobEmailParser()
    return parser.parse(email_text, verbose)