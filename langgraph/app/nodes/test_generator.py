import json
import uuid
from typing import Dict, Any, List, Optional
from langchain_google_genai import ChatGoogleGenerativeAI
from langchain_core.prompts import ChatPromptTemplate
from langchain_core.output_parsers import JsonOutputParser
from pydantic import BaseModel, Field
from dotenv import load_dotenv
from ..state import AgentState, TestJob

load_dotenv()

class RequestDetails(BaseModel):
    method: str = Field(description="HTTP Method")
    url_path: str = Field(description="URL path")
    headers: Optional[Dict[str, str]] = Field(default=None, description="HTTP Headers")
    body: Optional[Dict[str, Any]] = Field(default=None, description="JSON Body")
    params: Optional[Dict[str, Any]] = Field(default=None, description="Query Parameters")

class ExecutionJob(BaseModel):
    scenario_id: int
    setup_request: Optional[RequestDetails] = Field(default=None)
    test_request: RequestDetails

class JobBatch(BaseModel):
    jobs: List[ExecutionJob]

def test_generator_node(state: AgentState) -> Dict:
    print("🔨 Node: Test Generator - Building Stateful Payloads...")
    
    plans = state.get("test_plans", [])
    schema = state.get("global_schema", {})
    demo_data = state.get("demo_data", {})
    
    if not plans:
        return {"generated_jobs": []}

    try:
        llm = ChatGoogleGenerativeAI(model="gemini-2.5-flash", temperature=0.4, max_retries=3)
    except Exception:
        llm = ChatGoogleGenerativeAI(model="gemini-2.5-pro", temperature=0.4, max_retries=3)

    parser = JsonOutputParser(pydantic_object=JobBatch)

    prompt = ChatPromptTemplate.from_messages([
        ("system", "You are an API Test Data Generator. Convert test descriptions into exact HTTP requests."),
        ("user", """
        CONTEXT:
        API Schema: {schema}
        DEMO BLUEPRINTS: {demo_data} 
        
        TASK:
        Generate the JSON body, headers, and parameters for each scenario.

        CRITICAL STATEFULNESS RULES:
        1. USE THE BLUEPRINTS: When generating a POST or PUT body, use the realistic values provided in the DEMO BLUEPRINTS.
        2. If the test is for GET, PUT, PATCH, or DELETE, you MUST provide a 'setup_request' that is a valid POST request to create the resource first.
        3. In your 'test_request' url_path, use the exact string {{setup_id}} where the ID should go.
        4. If the test is a POST request, 'setup_request' should be null.

        TEST SCENARIOS:
        {plans}

        OUTPUT FORMAT:
        {format_instructions}
        """)
    ])

    chain = prompt | llm | parser

    try:
        batch_plans = plans[:5] 
        result = chain.invoke({
            "schema": json.dumps(schema, indent=2),
            "demo_data": json.dumps(demo_data, indent=2),
            "plans": json.dumps(batch_plans, indent=2),
            "format_instructions": parser.get_format_instructions()
        })
        
        raw_jobs = result.get("jobs", [])
        final_jobs: List[TestJob] = []
        
        for job in raw_jobs:
            final_jobs.append({
                "trace_id": str(uuid.uuid4()),
                "execution_instruction": {
                    "setup": job.get("setup_request"),
                    "test": job.get("test_request")
                },
                "pass_along_context": {
                    "original_plan_id": job.get("scenario_id")
                }
            })

        print(f"✅ Generator: Built {len(final_jobs)} stateful executable jobs.")
        return {
            "generated_jobs": final_jobs,
            "processing_logs": [f"Generated {len(final_jobs)} jobs"]
        }

    except Exception as e:
        print(f"❌ Generator Error: {e}")
        return {
            "generated_jobs": [],
            "processing_logs": [f"Generator Failed: {str(e)}"]
        }