import json
import os
from typing import Dict, Any

class Database:
    def __init__(self, connection_string: str = "local"):
        self.mode = connection_string
        self.file_path = "multi_tenant_db.json"
        if self.mode == "local" and not os.path.exists(self.file_path):
            with open(self.file_path, "w") as f:
                json.dump({}, f)

    def save_project_data(self, project_id: str, schema: Dict[str, Any], demo_data: Dict[str, Any]) -> bool:
        """Saves both the schema and the generated demo data for a tenant."""
        if self.mode == "local":
            with open(self.file_path, "r") as f:
                db = json.load(f)
            
            # Ensure project exists in DB
            if project_id not in db:
                db[project_id] = {}
                
            db[project_id]["schema"] = schema
            db[project_id]["demo_data"] = demo_data
            
            with open(self.file_path, "w") as f:
                json.dump(db, f, indent=2)
            return True
        return False

    def get_project_data(self, project_id: str) -> Dict[str, Any]:
        """Retrieves the full project profile."""
        if self.mode == "local":
            with open(self.file_path, "r") as f:
                db = json.load(f)
            return db.get(project_id, {"schema": {}, "demo_data": {}})
        return {"schema": {}, "demo_data": {}}

db_client = Database("local")