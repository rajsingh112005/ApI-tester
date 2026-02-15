from fastapi import FastAPI
from dotenv import load_dotenv
import uvicorn

load_dotenv()

app = FastAPI(title = "AI API TESTER")

@app.get("/")
def health_check():
    return {"status": "active", "service": "LangGraph Generator"}

if __name__ == "__main__":
    uvicorn.run(app , host="0.0.0.0" , port = 8000)