from typing import Dict , Any , List
from ..state import AgentState

def schema_understanding_node(state : AgentState) -> dict:
    print("Node Schema for understanding parsing document")

    raw_input = state.get("raw_input" , {})
    payload = raw_input.get("payload" , {})

    normalized_schema = {
        "api_title": payload.get("info", {}).get("title", "Unknown API"),
        "base_url": "/", 
        "endpoints": []
    }
    paths = payload.get("paths" , {})

    parsed_count = 0

    for path, methods in paths.items():
        for method, details in methods.items():
            if method.upper() in ["GET", "POST", "PUT", "DELETE", "PATCH"]:
                
                endpoint_info = {
                    "method": method.upper(),
                    "path": path,
                    "description": details.get("summary") or details.get("description", "No description"),
                    # In the future, we will extract params/body schema here
                }
                
                normalized_schema["endpoints"].append(endpoint_info)
                parsed_count += 1

    print(f"✅ Librarian: Successfully parsed {parsed_count} endpoints.")

    return {
        "global_schema": normalized_schema,
        "processing_logs": [f"Parsed {parsed_count} endpoints."]
    }