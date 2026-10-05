"""
Job Email Parser Package
A rule-based NLP system for extracting structured data from job application emails
"""

# Import individual modules
from .preprocessor import Preprocessor
from .company_extractor import CompanyExtractor
from .role_extractor import RoleExtractor
from .status_classifier import StatusClassifier
from .confidence_scorer import ConfidenceScorer
from .parser import JobEmailParser, parse_job_email


__version__ = '1.0.0'
__all__ = [
    'JobEmailParser',
    'parse_job_email',
    'Preprocessor',
    'CompanyExtractor',
    'RoleExtractor',
    'StatusClassifier',
    'ConfidenceScorer'
]