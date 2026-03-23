import json
from typing import Dict, Any
from langchain_google_genai import ChatGoogleGenerativeAI
from langchain_core.prompts import ChatPromptTemplate
from langchain_core.output_parsers import JsonOutputParser
from dotenv import load_dotenv
from ..state import AgentState

load_dotenv()

def data_seeder_node(state: AgentState) -> Dict:
    print("Node: Data Seeder - Inventing Demo Records...")
    
    schema = state.get("global_schema", {})
    if not schema or "endpoints" not in schema:
        print("Seeder: No schema found to seed data for.")
        return {"demo_data": {}, "processing_logs": ["Data Seeder skipped: No schema."]}

    # Find POST/PUT endpoints to figure out what data models exist
    models_to_seed = []
    for ep in schema.get("endpoints", []):
        if ep.get("method", "").upper() in ["POST", "PUT"]:
            models_to_seed.append(ep)

    if not models_to_seed:
        return {"demo_data": {}, "processing_logs": ["No POST/PUT endpoints found for seeding."]}

    try:
        llm = ChatGoogleGenerativeAI(model="gemini-2.5-flash", temperature=0.5, max_retries=3)
    except Exception:
        llm = ChatGoogleGenerativeAI(model="gemini-2.5-pro", temperature=0.5, max_retries=3)

    parser = JsonOutputParser()

    prompt = ChatPromptTemplate.from_messages([
        ("system", "You are a Mock Data Generator for an API Testing Framework."),
        ("user", """
        Analyze the following API endpoints to understand the required data structures.
        
        ENDPOINTS:
        {endpoints}

        TASK:
        Generate 1 realistic JSON payload for EACH of the provided endpoints.
        Do not use simple dummy text like "string". Use realistic names, emails, statuses, etc.

        Return ONLY a JSON object with a single key 'demo_records' containing a list of these objects.
        """)
    ])

    chain = prompt | llm | parser

    try:
        print("   Thinking... Gemini is writing mock data.")
        result = chain.invoke({
            "endpoints": json.dumps(models_to_seed, indent=2)
        })
        
        demo_records = result.get("demo_records", [])
        print(f"Seeder: Successfully created {len(demo_records)} demo records.")
        
        return {
            "demo_data": {"records": demo_records},
            "processing_logs": [f"Invented {len(demo_records)} demo records."]
        }

    except Exception as e:
        print(f"Seeder Error: {e}")
        return {
            "demo_data": {"records": []},
            "processing_logs": [f"Seeder Failed: {str(e)}"]
        }