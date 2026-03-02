from app.workflow import app

# This mimics the Swagger/OpenAPI file for your Blog API
blog_schema_payload = {
    "info": {"title": "My Awesome Blog API"},
    "paths": {
        "/users": {
            "post": {
                "summary": "Create a new user",
                "requestBody": {
                    "content": {
                        "application/json": {
                            "schema": {
                                "type": "object",
                                "properties": {
                                    "username": {"type": "string"},
                                    "uniqueid": {"type": "string"},
                                    "email": {"type": "string"},
                                    "profile_photo": {"type": "string"}
                                },
                                "required": ["username", "email", "uniqueid"]
                            }
                        }
                    }
                }
            }
        },
        "/blogs": {
            "post": {
                "summary": "Create a new blog post",
                "requestBody": {
                    "content": {
                        "application/json": {
                            "schema": {
                                "type": "object",
                                "properties": {
                                    "userid": {"type": "string"},
                                    "title": {"type": "string"},
                                    "content": {"type": "string"},
                                    "tags": {"type": "array", "items": {"type": "string"}}
                                },
                                "required": ["userid", "title", "content"]
                            }
                        }
                    }
                }
            }
        },
        "/comments": {
            "post": {
                "summary": "Add a comment to a blog",
                "requestBody": {
                    "content": {
                        "application/json": {
                            "schema": {
                                "type": "object",
                                "properties": {
                                    "blogid": {"type": "string"},
                                    "userid": {"type": "string"},
                                    "content": {"type": "string"}
                                },
                                "required": ["blogid", "userid", "content"]
                            }
                        }
                    }
                }
            }
        }
    }
}

# The Input to LangGraph
inputs = {
    "request_id": "ingest_blog_001",
    "project_id": "tenant_blog_project_123", # <--- Multi-tenant ID!
    "raw_input": {
        "request_id": "ingest_blog_001",
        "action_type": "INGEST_SCHEMA",      # <--- Triggers the Learning Branch
        "payload": blog_schema_payload
    }
}

print("🚀 Starting Blog Schema Ingestion...")

try:
    result = app.invoke(inputs) # type: ignore
    print("\n✅ Execution Finished!")
    for log in result.get("processing_logs", []):
        print(f"   - {log}")
except Exception as e:
    print(f"❌ Execution Error: {e}")