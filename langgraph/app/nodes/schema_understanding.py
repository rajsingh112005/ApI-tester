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
    route_groups = payload.get("routes", {})

    parsed_count = 0

    # 1) OpenAPI-style payload: payload.paths.{path}.{method}
    if isinstance(paths, dict) and paths:
        for path, methods in paths.items():
            if not isinstance(methods, dict):
                continue

            for method, details in methods.items():
                method_upper = str(method).upper()
                if method_upper in ["GET", "POST", "PUT", "DELETE", "PATCH"]:
                    details = details if isinstance(details, dict) else {}

                    endpoint_info = {
                        "method": method_upper,
                        "path": path,
                        "description": details.get("summary") or details.get("description", "No description"),
                    }

                    normalized_schema["endpoints"].append(endpoint_info)
                    parsed_count += 1

    # 2) Extractor-style payload: payload.routes.routes = [{method, path, ...}]
    elif isinstance(route_groups, dict):
        extracted_routes = route_groups.get("routes", [])
        if isinstance(extracted_routes, list):
            for route in extracted_routes:
                if not isinstance(route, dict):
                    continue

                method_upper = str(route.get("method", "")).upper()
                path = route.get("path", "")

                if method_upper in ["GET", "POST", "PUT", "DELETE", "PATCH"] and path:
                    endpoint_info = {
                        "method": method_upper,
                        "path": path,
                        "description": route.get("handlerName") or "No description",
                        "handler_name": route.get("handlerName"),
                        "schema": route.get("schema", {}),
                    }

                    normalized_schema["endpoints"].append(endpoint_info)
                    parsed_count += 1

    print(f"✅ Librarian: Successfully parsed {parsed_count} endpoints.")

    return {
        "global_schema": normalized_schema,
        "processing_logs": [f"Parsed {parsed_count} endpoints."]
    }