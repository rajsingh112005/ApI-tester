import json
import os
from ..state import AgentState

DB_FILE = "db_mock.json"

def update_global_state_node(state: AgentState) -> dict:
    print("💾 Node: Update Global State - Saving to Database...")
    
    # 1. Get the new schema from the previous node
    new_schema = state.get("global_schema")
    
    if not new_schema:
        print("Archivist: No schema data found to save.")
        return {"processing_logs": ["No schema to save."]}

    try:
        current_db = {}
        if os.path.exists(DB_FILE):
            with open(DB_FILE, "r") as f:
                try:
                    current_db = json.load(f)
                except json.JSONDecodeError:
                    current_db = {}
        
        
        current_db["latest_project_schema"] = new_schema
        
        # Save back to file
        with open(DB_FILE, "w") as f:
            json.dump(current_db, f, indent=2)
            
        print(" Archivist: Knowledge Base Updated Successfully.")
        
    except Exception as e:
        print(f" Archivist Error: DB Write failed. {e}")
        return {"processing_logs": [f"DB Write Error: {e}"]}

    return {
        "processing_logs": ["Global Schema saved to db_mock.json"]
    }