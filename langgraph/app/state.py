from typing import TypedDict , List , Dict , Any , Annotated
import operator

#used for passing messages along the messaging queue
class TestJob(TypedDict):
    trace_id: str
    execution_instruction: Dict[str , Any]
    pass_along_context: Dict[str , Any]

#main graph states
class AgentState(TypedDict):
    request_id: str
    project_id: str
    raw_input: Dict[str , Any]
    # determining which node to take
    route_decision: str

    #global knolwedge
    global_schema: Dict[str , Any]
    demo_data: Dict[str , Any]
    test_plans: List[Dict[str , Any]]
    # operator.add ensures we append to logs/jobs instead of overwriting
    generated_jobs: Annotated[List[TestJob], operator.add]
    processing_logs: Annotated[List[str], operator.add]

