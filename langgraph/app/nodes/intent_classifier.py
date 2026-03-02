from typing import Dict , Any , Literal
from pydantic import BaseModel , ValidationError
from ..state import AgentState

# validation model
class RequestPayload(BaseModel):
    request_id: str
    action_type: Literal["DETECT_AUTO", "INGEST_SCHEMA", "GENERATE_TEST"] = "DETECT_AUTO"
    payload: Dict[str , Any]

#node logic for intent classification
def intent_classifier_node(state: AgentState) -> dict:
    print("Node: Intent Classifier - Analyzing Request...")
    
    raw_input = state.get("raw_input", {})
    
    try:
        request = RequestPayload(**raw_input)
    except ValidationError as e:
        print(f" Validation Error: {e}")
        
        return {
            "route_decision": "error",
            "processing_logs": [f"Input validation failed: {e}"]
        }
    
    route_decision = ""
    logs=[]

    if request.action_type == "INGEST_SCHEMA":
        route_decision = "schema_path"
        logs.append("Action: INGEST_SCHEMA detected.")
        
    elif request.action_type == "GENERATE_TEST":
        route_decision = "test_path"
        logs.append("Action: GENERATE_TEST detected.")
        
    else:
        payload_str = str(request.payload).lower()
        if "openapi" in payload_str or "swagger" in payload_str:
            route_decision = "schema_path"
            logs.append("Auto-detected Schema content.")
        else:
            route_decision = "test_path"
            logs.append("Auto-detected Test Generation request.")

    return {
        "request_id": request.request_id,
        "route_decision": route_decision,
        "processing_logs": logs
    }