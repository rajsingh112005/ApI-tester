from app.workflow import app
from  app.state import AgentState 

# Simulate a user pasting a Swagger/OpenAPI JSON
mock_input = {
    "request_id": "test_001",
    "action_type": "INGEST_SCHEMA", 
    "payload": {
        "openapi": "3.0.0",
        "info": {"title": "My Test API", "version": "1.0.0"},
        "paths": {
            "/users": {
                "post": {"summary": "Create a user"},
                "get": {"summary": "List users"}
            },
            "/orders": {
                "get": {"summary": "List orders"}
            }
        }
    }
}

print("🚀 Starting Learning Graph Test...\n")

# Run the Graph
initial_state: AgentState = {
    "request_id": mock_input["request_id"],
    "raw_input": mock_input,
    "route_decision": "",
    "global_schema": {},
    "test_plans": [],
    "generated_jobs": [],
    "processing_logs": []
}

result = app.invoke(initial_state)

print("\n✅ Execution Finished!")
print("Final Logs:", result["processing_logs"])
print("Global Schema:", result.get("global_schema", {}))