import json
import uuid
import pika
from typing import Any, Dict, List
from ..state import AgentState

def mq_producer_node(state: AgentState) -> Dict:
    print("MQ Producer - pushing tests to queues")

    jobs = state.get("generated_jobs", [])
    logs = []

    if not jobs:
        print("No jobs to push")
        return {"processing_logs: ": ["No jobs to push"]}
    
    batch_id = str(uuid.uuid4())
    print(f"Batch ID: {batch_id} - Pushing {len(jobs)} jobs to MQ")

    try:

        connection = pika.BlockingConnection(pika.ConnectionParameters('localhost'))
        channel = connection.channel()
        queue_name = 'test_jobs_queue'
        channel.queue_declare(queue=queue_name, durable=True)

        for job in jobs:
            # 3. Standardize the Message
            mq_message = {
                "message_id": job.get("trace_id"),
                "batch_id": batch_id,
                "priority": "high",
                "action": "EXECUTE_TEST",
                "payload": job.get("execution_instruction"), 
                "metadata": job.get("pass_along_context")
            }
            
            # 4. Publish to RabbitMQ
            channel.basic_publish(
                exchange='',
                routing_key=queue_name,
                body=json.dumps(mq_message),
                properties=pika.BasicProperties(
                    delivery_mode=2,
                )
            )
            
            test_req = mq_message['payload'].get('test') or {}
            method = test_req.get('method', 'UNKNOWN')
            url = test_req.get('url_path', 'UNKNOWN_URL')
            
            print(f"   ➡️ [RabbitMQ] Sent: Test for {method} {url}")
            logs.append(f"Pushed Job {job['trace_id']} to RabbitMQ")

        connection.close()
        print(f"Successfully pushed {len(jobs)} messages to RabbitMQ.")
        
    except Exception as e:
        print(f"MQ Connection Error: {e}")
        logs.append(f"MQ Error: {e}")

    return {
        "processing_logs": logs
    }