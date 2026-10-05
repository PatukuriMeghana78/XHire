"""

Demonstrates the parser with various test cases


"""

import sys
import os

# Add src directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

# Now import from src
from parser import JobEmailParser

def print_result(email_description, result, show_details=False):
    """Pretty print parsing results."""
    print(f"\n{'='*70}")
    print(f"TEST: {email_description}")
    print('='*70)
    print(f"Company:    {result['company']}")
    print(f"Role:       {result['role']}")
    print(f"Status:     {result['status']}")
    print(f"Confidence: {result['confidence']}%")
    
    if show_details and 'details' in result:
        print(f"\n--- Detailed Breakdown ---")
        print(f"Company Method: {result['details']['company_details']['method']}")
        print(f"Role Method: {result['details']['role_details']['method']}")
        print(f"Status Keywords Found: {len(result['details']['status_details']['keyword_matches'][result['status']])} matches")
        print(f"\nExplanation:\n{result['details']['explanation']}")

def main():
    # Initialize parser
    parser = JobEmailParser()
    
    print("="*70)
    print("JOB EMAIL PARSER - DEMONSTRATION")
    print("Rule-Based NLP Capstone Project")
    print("="*70)
    
    # Test Case 1: Application Confirmation
    email1 = """
    Subject: Application Received - Software Engineer
    
    Dear Applicant,
    
    Thank you for applying to the Software Engineer position at TechCorp.
    We have received your application and our team will review it shortly.
    
    Best regards,
    TechCorp Recruiting Team
    careers@techcorp.com
    """
    result1 = parser.parse(email1, verbose=True)
    print_result("Application Confirmation Email", result1, show_details=True)
    
    # Test Case 2: Interview Invitation
    email2 = """
    From: hiring@google.com
    Subject: Interview Request - Senior Data Analyst
    
    Hi there,
    
    We were impressed with your application for the Senior Data Analyst position
    at Google. We would like to schedule an interview with you next week to discuss 
    your experience further. Are you available for a video call on Tuesday?
    
    Looking forward to speaking with you!
    
    Best,
    Google Recruiting Team
    """
    result2 = parser.parse(email2, verbose=False)
    print_result("Interview Invitation", result2)
    
    # Test Case 3: Rejection Email
    email3 = """
    Subject: Re: Product Manager Application
    
    Dear Candidate,
    
    Thank you for your interest in the Product Manager role at Amazon.
    
    Unfortunately, we have decided to move forward with other candidates
    whose experience more closely matches our current needs. We appreciate
    the time you invested in the application process.
    
    We wish you the best in your job search.
    
    Amazon Talent Team
    talent@amazon.com
    """
    result3 = parser.parse(email3, verbose=False)
    print_result("Rejection Email", result3)
    
    # Test Case 4: Mixed Signals (Applied + Interview)
    email4 = """
    Subject: Your Application Update
    
    Hello,
    
    Thank you for applying to the Frontend Developer position at Meta.
    We received your application and are impressed with your background.
    We'd like to schedule a phone screen with you to discuss the role further.
    
    Please let us know your availability.
    
    Meta Recruiting
    """
    result4 = parser.parse(email4, verbose=True)
    print_result("Mixed Signals Email", result4, show_details=True)
    
    # Test Case 5: Informal Email
    email5 = """
    hey!
    
    saw your profile and think you'd be great for our ML Engineer role at DataCo.
    wanna hop on a call sometime this week to chat about it?
    
    let me know!
    sarah
    recruiting@dataco.ai
    """
    result5 = parser.parse(email5, verbose=False)
    print_result("Informal/Casual Email", result5)
    
    # Test Case 6: Automated/Structured
    email6 = """
    [AUTOMATED MESSAGE]
    
    Application ID: #67890
    Position: Backend Engineer
    Company: Stripe
    Status: Application Successfully Submitted
    
    Your application is under review. You will hear from us within 2 weeks
    if your profile matches our requirements.
    """
    result6 = parser.parse(email6, verbose=False)
    print_result("Automated Confirmation", result6)
    
    # Test Case 7: Take-home Assignment (Interview stage)
    email7 = """
    Subject: Coding Challenge - Full Stack Engineer Position
    
    Hello,
    
    Great speaking with you! As discussed, here's the take-home coding challenge
    for the Full Stack Engineer position at Shopify. Please complete and submit
    within 5 days.
    
    [Assignment details...]
    
    Looking forward to reviewing your submission!
    
    Shopify Engineering Team
    """
    result7 = parser.parse(email7, verbose=False)
    print_result("Take-home Assignment", result7)
    
    # Test Case 8: Position Filled
    email8 = """
    Dear Applicant,
    
    Thank you for your application for the UX Designer role at Adobe.
    We wanted to inform you that this position has been filled. We encourage
    you to apply for other openings that match your skills.
    
    Best wishes,
    Adobe Talent Acquisition
    """
    result8 = parser.parse(email8, verbose=False)
    print_result("Position Filled (Rejection)", result8)
    
    # Summary Statistics
    print(f"\n{'='*70}")
    print("SUMMARY STATISTICS")
    print('='*70)
    
    results = [result1, result2, result3, result4, result5, result6, result7, result8]
    
    # Count by status
    status_counts = {}
    for r in results:
        status_counts[r['status']] = status_counts.get(r['status'], 0) + 1
    
    print(f"\nStatus Distribution:")
    for status, count in status_counts.items():
        print(f"  {status}: {count}")
    
    # Average confidence
    avg_confidence = sum(r['confidence'] for r in results) / len(results)
    print(f"\nAverage Confidence: {avg_confidence:.1f}%")
    
    # Extraction success rates
    company_success = sum(1 for r in results if r['company'] is not None)
    role_success = sum(1 for r in results if r['role'] is not None)
    
    print(f"\nExtraction Success Rates:")
    print(f"  Company: {company_success}/{len(results)} ({company_success/len(results)*100:.0f}%)")
    print(f"  Role: {role_success}/{len(results)} ({role_success/len(results)*100:.0f}%)")
    print(f"  Status: {len(results)}/{len(results)} (100%)")
    
    print(f"\n{'='*70}")
    print("DEMO COMPLETE")
    print("="*70)

if __name__ == "__main__":
    main()