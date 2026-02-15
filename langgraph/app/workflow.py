from typing import Literal
from langgraph.graph import StateGraph , END

from .state import AgentState
from .nodes.intent_classifier import intent_classifier_node
from .nodes.schema_understanding import schema_understanding_node
from .nodes.update_global_state import update_global_state_node

def route_workflow(state: AgentState) -> Literal["schema_understanding" , "end"]:
    decision = state.get("route_decision" , "")

    if decision == "schema_path":
        return "schema_understanding"
    else:
        print("wait")
        return "end"
    
workflow = StateGraph(AgentState)

workflow.add_node("intent_classifier" , intent_classifier_node)
workflow.add_node("schema_understanding" , schema_understanding_node)
workflow.add_node("update_global_state" , update_global_state_node)

workflow.set_entry_point("intent_classifier")

workflow.add_conditional_edges(
    "intent_classifier",
    route_workflow,
    {
        "schema_understanding": "schema_understanding",
        "end": END
    }
)

workflow.add_edge("schema_understanding", "update_global_state")
workflow.add_edge("update_global_state", END)

app = workflow.compile()