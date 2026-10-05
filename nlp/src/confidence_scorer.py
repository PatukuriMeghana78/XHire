"""
Confidence Scorer Module
Calculates overall confidence for the extraction results
"""

class ConfidenceScorer:
    """
    Aggregates confidence from all extraction modules.
    
    Uses weighted average based on field reliability.
    """
    
    def __init__(self):
        # Weights for different fields (must sum to 1.0)
        self.weights = {
            'company': 0.30,   # 30% - moderate importance
            'role': 0.30,      # 30% - moderate importance
            'status': 0.40     # 40% - highest importance (affects user action)
        }
    
    def calculate_overall_confidence(self, company_conf: int, role_conf: int, 
                                    status_conf: int) -> dict:
        """
        Calculate weighted overall confidence score.
        
        Args:
            company_conf: Confidence score for company (0-100)
            role_conf: Confidence score for role (0-100)
            status_conf: Confidence score for status (0-100)
        
        Returns:
            {
                'overall': int (weighted average),
                'breakdown': dict (individual scores),
                'reliability': str (High/Medium/Low),
                'weights_used': dict
            }
        """
        # Calculate weighted average
        overall = (
            company_conf * self.weights['company'] +
            role_conf * self.weights['role'] +
            status_conf * self.weights['status']
        )
        
        overall = round(overall)
        
        # Determine reliability category
        if overall >= 80:
            reliability = 'High'
        elif overall >= 60:
            reliability = 'Medium'
        else:
            reliability = 'Low'
        
        return {
            'overall': overall,
            'breakdown': {
                'company': company_conf,
                'role': role_conf,
                'status': status_conf
            },
            'reliability': reliability,
            'weights_used': self.weights.copy()
        }
    
    def adjust_for_missing_fields(self, company_result: dict, 
                                  role_result: dict, status_result: dict) -> int:
        """
        Adjust confidence when fields are missing.
        
        Missing critical information reduces overall confidence.
        """
        penalties = 0
        
        # Company missing - moderate penalty
        if company_result.get('company') is None:
            penalties += 10
        
        # Role missing - moderate penalty
        if role_result.get('role') is None:
            penalties += 10
        
        # Status is never truly "missing" (defaults to Applied)
        # But low status confidence is already factored in
        
        return penalties
    
    def get_confidence_explanation(self, company_result: dict, role_result: dict, 
                                   status_result: dict, overall: int) -> str:
        """
        Generate human-readable explanation of confidence score.
        
        Useful for debugging and viva demonstration.
        """
        explanations = []
        
        # Company explanation
        if company_result['company']:
            explanations.append(
                f"Company '{company_result['company']}' found via "
                f"{company_result['method']} ({company_result['confidence']}% confident)"
            )
        else:
            explanations.append("Company not detected (reduces confidence)")
        
        # Role explanation
        if role_result['role']:
            explanations.append(
                f"Role '{role_result['role']}' found via "
                f"{role_result['method']} ({role_result['confidence']}% confident)"
            )
        else:
            explanations.append("Job role not detected (reduces confidence)")
        
        # Status explanation
        status_matches = status_result.get('keyword_matches', {})
        matched_keywords = status_matches.get(status_result['status'], [])
        
        if matched_keywords:
            explanations.append(
                f"Status '{status_result['status']}' classified based on "
                f"{len(matched_keywords)} keyword(s): {', '.join(matched_keywords[:3])}"
                f" ({status_result['confidence']}% confident)"
            )
        else:
            explanations.append(
                f"Status defaulted to '{status_result['status']}' "
                f"(no clear keywords, {status_result['confidence']}% confident)"
            )
        
        # Overall
        explanations.append(f"\nOverall confidence: {overall}%")
        
        return "\n".join(explanations)