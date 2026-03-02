from typing import Literal
from langgraph.graph import StateGraph, END

from .state import AgentState
from .nodes.intent_classifier import intent_classifier_node
from .nodes.schema_understanding import schema_understanding_node
from .nodes.data_seeder import data_seeder_node         # <--- NEW IMPORT
from .nodes.update_global_state import update_global_state_node
from .nodes.strategy_agent import strategy_agent_node
from .nodes.test_generator import test_generator_node
from .nodes.mq_producer import mq_producer_node

def route_workflow(state: AgentState) -> Literal["schema_understanding", "strategy_agent", "end"]:
    decision = state.get("route_decision", "")

    if decision == "schema_path":
        return "schema_understanding"
    elif decision == "test_path":
        return "strategy_agent"
    else:
        return "end"
    
workflow = StateGraph(AgentState)

# Add Nodes
workflow.add_node("intent_classifier", intent_classifier_node)
workflow.add_node("schema_understanding", schema_understanding_node)
workflow.add_node("data_seeder", data_seeder_node)              
workflow.add_node("update_global_state", update_global_state_node)
workflow.add_node("strategy_agent", strategy_agent_node)
workflow.add_node("test_generator", test_generator_node)
workflow.add_node("mq_producer", mq_producer_node)

# Set Entry
workflow.set_entry_point("intent_classifier")

# Conditional Edges
workflow.add_conditional_edges(
    "intent_classifier",
    route_workflow,
    {
        "schema_understanding": "schema_understanding",
        "strategy_agent": "strategy_agent",
        "end": END
    }
)

# LEARNING BRANCH (Schema Ingestion -> Seeding -> Saving)
workflow.add_edge("schema_understanding", "data_seeder")         # <--- Parse schema, then invent data
workflow.add_edge("data_seeder", "update_global_state")          # <--- Save both to DB
workflow.add_edge("update_global_state", END)

#TESTING BRANCH (Planning -> Generating -> Queuing)
workflow.add_edge("strategy_agent", "test_generator")
workflow.add_edge("test_generator", "mq_producer")
workflow.add_edge("mq_producer", END)

app = workflow.compile()