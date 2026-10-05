from nlp.src.email_classification_agent import EmailClassificationAgent

agent = EmailClassificationAgent()

email = """
Subject: Interview Invitation

Dear Candidate,

We would like to schedule a technical round interview for the position of Data Analyst at Amazon.

Please confirm your availability for a Zoom meeting tomorrow.

Best,
Amazon Hiring Team
"""

result = agent.classify(email)

print(result)