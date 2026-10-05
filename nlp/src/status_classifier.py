"""
Status Classifier Module
Classifies application status using keyword scoring
"""

class StatusClassifier:
    """
    Classifies email into: Applied, Interview, or Rejected.
    
    Method: Keyword scoring with priority weighting
    """
    
    def __init__(self):
        # Status-specific keywords (lowercase)
        self.keywords = {
            'Applied': [
                'application received',
                'thank you for applying',
                'we received your application',
                'application submitted',
                'successfully applied',
                'application confirmation',
                'we have received',
                'received your resume',
                'under review',
                'reviewing your application',
                'application is being reviewed',
            ],
            
            'Interview': [
                'interview',
                'schedule a call',
                'schedule a meeting',
                'would like to speak',
                'would like to talk',
                'next steps',
                'move forward',
                'moving forward',
                'screening call',
                'phone screen',
                'video call',
                'meeting',
                'available to chat',
                'discuss your application',
                'talk with you',
                'invitation to interview',
                'schedule time',
                'calendar',
                'availability',
                'meet with',
                'speak with you',
            ],
            
            'Rejected': [
                'unfortunately',
                'regret to inform',
                'regret to tell',
                'not moving forward',
                'not selected',
                'other candidates',
                'another candidate',
                'position has been filled',
                'position is filled',
                'decided to pursue',
                'will not be moving',
                'not a match',
                'not the right fit',
                'unable to offer',
                'declined',
                'chosen to move forward with',
                'pursuing other',
                'decided not to move forward',
                'not proceeding',
            ],
            'Offer': [ 
                'pleased to offer',
                'extend an offer',
                'offer letter',
                'accept our offer',
                'congratulations',
                'offer of employment',
                'compensation package'
    ]
        }
        
        # Priority multipliers (Interview and Rejection are stronger signals)
        self.multipliers = {
            'Applied': 1.0,
            'Interview': 1.5,   # Interview signals are strong
            'Rejected': 1.3     # Rejection signals are strong
        }
    
    def classify(self, lowercase_text: str) -> dict:
        """
        Classify status with confidence score.
        
        Returns:
            {
                'status': str ('Applied', 'Interview', or 'Rejected'),
                'confidence': int (0-100),
                'keyword_matches': dict (keywords found per category)
            }
        """
        # Count keyword matches for each status
        scores = {status: 0 for status in self.keywords}
        matches = {status: [] for status in self.keywords}
        
        for status, keyword_list in self.keywords.items():
            for keyword in keyword_list:
                count = lowercase_text.count(keyword)
                if count > 0:
                    scores[status] += count
                    matches[status].append(keyword)
        
        # Apply priority multipliers
        for status in scores:
            scores[status] *= self.multipliers[status]
        
        # Find winning status
        max_score = max(scores.values())
        
        if max_score == 0:
            # No keywords found - default to 'Applied' with low confidence
            return {
                'status': 'Applied',
                'confidence': 50,  # Low confidence default
                'keyword_matches': matches,
                'scores': scores
            }
        
        # Get status with highest score
        winning_status = max(scores.keys(), key=lambda k: scores[k])
        
        # Calculate confidence based on score and matches
        confidence = self._calculate_confidence(
            winning_status, 
            scores[winning_status],
            len(matches[winning_status]),
            max_score
        )
        
        return {
            'status': winning_status,
            'confidence': confidence,
            'keyword_matches': matches,
            'scores': scores
        }
    
    def _calculate_confidence(self, status: str, score: float, 
                            match_count: int, max_score: float) -> int:
        """
        Calculate confidence score (0-100) based on evidence strength.
        
        Factors:
        - Number of keyword matches
        - Score relative to other categories
        - Status type (Rejected is most reliable)
        """
        # Base confidence from match count
        if match_count >= 3:
            base = 95
        elif match_count == 2:
            base = 85
        elif match_count == 1:
            base = 75
        else:
            base = 50
        
        # Boost for rejection (very distinctive keywords)
        if status == 'Rejected' and match_count >= 1:
            base = min(base + 10, 100)
        
        # Boost for interview (distinctive keywords)
        if status == 'Interview' and match_count >= 2:
            base = min(base + 5, 100)
        
        # Reduce if score is close to other categories (ambiguous)
        other_scores = [s for s in [score] if s != max_score]
        if other_scores and max(other_scores) / max_score > 0.7:
            # Close competition - reduce confidence
            base -= 15
        
        return max(min(base, 100), 0)  # Clamp to [0, 100]