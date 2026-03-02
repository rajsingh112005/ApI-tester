import json
import os
from typing import Dict, Any, List

# If you haven't installed this yet, run: pip install langchain-google-genai
from langchain_google_genai import ChatGoogleGenerativeAI
from langchain_core.prompts import ChatPromptTemplate
from langchain_core.output_parsers import JsonOutputParser
from pydantic import BaseModel, Field
from dotenv import load_dotenv

from ..state import AgentState

load_dotenv()

class TestScenario(BaseModel):
    test_type: str = Field(description="Category: POSITIVE, NEGATIVE, BOUNDARY, SECURITY")
    description: str = Field(description="Brief summary of what to verify")
    expected_status: int = Field(description="The expected HTTP status code")

class TestPlan(BaseModel):
    scenarios: List[TestScenario]

def strategy_agent_node(state: AgentState) -> Dict:
    print("Node: Strategy Agent - Planning Tests...")

    raw_input = state.get("raw_input", {})
    payload = raw_input.get("payload", raw_input) 
    target_endpoint_str = payload.get("target_endpoint", "")
    
    global_schema = state.get("global_schema", {})
    endpoint_details = {}

    endpoint_details = {}
    if global_schema and "endpoints" in global_schema:
        try:
            target_method, target_path = target_endpoint_str.split(" ", 1)
        except ValueError:
            target_method, target_path = "GET", target_endpoint_str

        for ep in global_schema["endpoints"]:
            schema_method = ep.get('method', '').upper()
            schema_path = ep.get('path', '').lower()
            if schema_method == target_method.upper() and schema_path == target_path.lower():
                endpoint_details = ep
                break
    
    if not endpoint_details:
        print(f" Warning: No schema found for {target_endpoint_str}. Guessing logic.")
    else:
        print(f" Context: Found schema for {target_endpoint_str}")

    if not os.getenv("GOOGLE_API_KEY"):
         return {"processing_logs": [" Error: GOOGLE_API_KEY not found in .env"]}

    try:
        llm = ChatGoogleGenerativeAI(
            model="gemini-2.5-flash",
            temperature=0.2
        )
    except Exception as e:
        print(f"LLM Setup Error: {e}")
        return {"processing_logs": ["LLM Setup Failed"]}

    parser = JsonOutputParser(pydantic_object=TestPlan)
    
    prompt = ChatPromptTemplate.from_messages([
        ("system", "You are an expert QA Automation Lead. Create a comprehensive API test strategy."),
        ("user", """
        Analyze the following API Endpoint definition and generate a list of distinct test scenarios.

        TARGET ENDPOINT: {target}
        SCHEMA CONTEXT: {schema}

        CRITICAL HTTP METHOD INSTRUCTIONS:
        - If GET: Focus on query parameters, path variables (e.g. /users/{{id}}), and data retrieval. No body payloads.
        - If POST: Focus on payload validation, missing required fields, and resource creation.
        - If PUT/PATCH: Focus on partial/full payload updates and replacing resources via path variables.
        - If DELETE: Focus on resource removal, idempotency, and 404s for already deleted items.

        Include Happy Path, Data Validation (Negative), and Security checks.
        Return ONLY a JSON object with a 'scenarios' key.
        Format requirements:
        {format_instructions}
        """)
    ])
    
    chain = prompt | llm | parser
    
    try:
        print(f"   Thinking... Gemini is planning for {target_endpoint_str}")
        
        result = chain.invoke({
            "target": target_endpoint_str,
            "schema": json.dumps(endpoint_details, indent=2),
            "format_instructions": parser.get_format_instructions()
        })
        
        # Extract scenarios
        generated_plans = result.get("scenarios", [])
        
        print(f"Planner: Gemini devised {len(generated_plans)} test scenarios.")
        
        return {
            "test_plans": generated_plans,
            "processing_logs": [f"AI Generated {len(generated_plans)} scenarios"]
        }

    except Exception as e:
        print(f"Planner Error: AI generation failed. {e}")
        return {
            "test_plans": [],
            "processing_logs": [f"Strategy Error: {str(e)}"]
        }