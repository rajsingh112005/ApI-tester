from fastapi import FastAPI , HTTPException
from pydantic import BaseModel
from typing import Dict , Any
import uuid

from app.workflow import app as langgraph_app
from app.db import db_client

app = FastAPI(title="AI API Tester Platform" , version="1.0")

class SchemaIngestRequest(BaseModel):
    project_id: str
    schema_payload: Dict[str , Any]

class TestGenerateRequest(BaseModel):
    project_id: str
    target_endpoint: str

@app.post("/api/v1/projects/ingest")
async def ingest_schema(request: SchemaIngestRequest):
    """Triggers the Learning Branch: Parses schema and generates demo data."""
    req_id = f"ingest_{uuid.uuid4().hex[:8]}"
    
    inputs = {
        "request_id": req_id,
        "project_id": request.project_id,
        "raw_input": {
            "request_id": req_id,
            "action_type": "INGEST_SCHEMA",
            "payload": request.schema_payload
        }
    }
    
    try:
        result = langgraph_app.invoke(inputs) # type: ignore
        return {
            "status": "success",
            "project_id": request.project_id,
            "message": "Schema ingested and demo data seeded successfully.",
            "logs": result.get("processing_logs", [])
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/api/v1/projects/generate-tests")
async def generate_tests(request: TestGenerateRequest):
    """Triggers the Testing Branch: Generates tests and queues them in RabbitMQ."""
    
    project_data = db_client.get_project_data(request.project_id)
    schema = project_data.get("schema")
    demo_data = project_data.get("demo_data")
    
    if not schema:
        raise HTTPException(status_code=404, detail="Project schema not found in DB. Ingest it first.")

    req_id = f"test_{uuid.uuid4().hex[:8]}"
    inputs = {
        "request_id": req_id,
        "project_id": request.project_id,
        "raw_input": {
            "request_id": req_id,
            "action_type": "GENERATE_TEST",
            "payload": {"target_endpoint": request.target_endpoint}
        },
        "route_decision": "test_path",
        "global_schema": schema,
        "demo_data": demo_data
    }

    try:
        # 3. Run the AI Generation
        result = langgraph_app.invoke(inputs) # type: ignore
        jobs = result.get("generated_jobs", [])
        
        return {
            "status": "success",
            "message": f"Successfully generated {len(jobs)} tests and pushed to RabbitMQ.",
            "jobs_queued": len(jobs),
            "logs": result.get("processing_logs", [])
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))