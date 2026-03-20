import json
import os
from typing import Dict, Any
from pymongo import MongoClient
from dotenv import load_dotenv

load_dotenv()

class Database:
    def __init__(self):
        self.mongo_uri = os.getenv("MONGO_URI")

        if not self.mongo_uri:
            print("Mongo_URI not found in env file")
            self.mongo_uri = "mongodb://localhost:27017/"
        
        self.client = MongoClient(self.mongo_uri)

        self.db = self.client["api_tester_platform"]
        self.collection = self.db["blueprints"]

    def save_project_data(self, project_id: str, schema: Dict[str, Any], demo_data: Dict[str, Any]) -> bool:
        """Upserts the schema and demo data into MongoDB Atlas."""
        try:
            self.collection.update_one(
                {"project_id": project_id},
                {"$set": {"schema": schema, "demo_data": demo_data}},
                upsert=True
            )
            return True
        except Exception as e:
            print(f"MongoDB Write Error: {e}")
            return False

    def get_project_data(self, project_id: str) -> Dict[str, Any]:
        """Fetches the project blueprint from MongoDB in milliseconds."""
        try:
            document = self.collection.find_one({"project_id": project_id})
            if document:
                return {
                    "schema": document.get("schema", {}), 
                    "demo_data": document.get("demo_data", {})
                }
        except Exception as e:
            print(f"MongoDB Read Error: {e}")
            
        return {"schema": {}, "demo_data": {}}

db_client = Database()