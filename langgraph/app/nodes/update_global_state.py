import json
import os
from ..state import AgentState
from app.db import db_client

DB_FILE = "db_mock.json"

def update_global_state_node(state: AgentState) -> dict:
    print("Node: Update Global State - Saving to Database...")
    
    project_id = state.get("project_id", "default_project") # Fallback for now
    new_schema = state.get("global_schema", {})
    demo_data = state.get("demo_data", {})
    
    if not new_schema:
        print("Archivist: No schema data found to save.")
        return {"processing_logs": ["No schema to save."]}

    try:
        # Save both schema and demo data using the updated DB client
        db_client.save_project_data(project_id, new_schema, demo_data)
        print(f"Archivist: Knowledge Base & Demo Data updated for '{project_id}'.")
        
        return {
            "processing_logs": [f"Saved schema and {len(demo_data.get('records', []))} mock records to DB."]
        }
        
    except Exception as e:
        print(f"Archivist Error: DB Write failed. {e}")
        return {"processing_logs": [f"DB Write Error: {e}"]}