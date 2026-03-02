import json
from app.workflow import app
from app.db import db_client

target = "PUT /blogs/{blogid}"  # Testing an update to a blog post!
project_id = "tenant_blog_project_123"

print(f"\n🚀 Loading context for {project_id}...")

# 1. Fetch the specific Tenant's context from our DB
project_data = db_client.get_project_data(project_id)
schema = project_data.get("schema", {})
demo_data = project_data.get("demo_data", {})

if not schema:
    print("❌ Error: No schema found for this project. Run ingest_blog.py first.")
    exit()

# 2. Inject it into the Graph
inputs = {
    "request_id": "test_blog_001",
    "project_id": project_id,
    "raw_input": {
        "request_id": "test_blog_001",
        "action_type": "GENERATE_TEST",
        "payload": {"target_endpoint": target}
    },
    "route_decision": "test_path",
    "global_schema": schema,
    "demo_data": demo_data  # <--- Passing the blueprints!
}

print(f"🎯 Starting Generation for: {target}...\n")

try:
    result = app.invoke(inputs) # type: ignore
    
    print("\n✅ Execution Finished!")
    print("-" * 50)
    
    jobs = result.get("generated_jobs", [])
    print(f"\n🔨 Builder: {len(jobs)} Executable Jobs Ready")
    
    for i, job in enumerate(jobs, 1):
        instr = job.get("execution_instruction", {})
        setup = instr.get("setup")
        test = instr.get("test")
        
        print(f"\n   🔹 Job #{i} (Trace: {job.get('trace_id')[:8]}...)")
        
        # Safely print SETUP
        if setup:
            print(f"      [SETUP] {setup.get('method')} {setup.get('url_path')}")
            if setup.get('body'):
                print(f"              Body: {setup.get('body')}")
        
        # Safely print TEST
        if test:
            print(f"      [TEST]  {test.get('method')} {test.get('url_path')}")
            if test.get('body'):
                print(f"              Body: {test.get('body')}")
        else:
            print("      [TEST]  Failed to generate valid test payload.")

except Exception as e:
    print(f"❌ Execution Error: {e}")