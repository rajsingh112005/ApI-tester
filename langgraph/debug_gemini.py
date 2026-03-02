import os
import google.generativeai as genai
from dotenv import load_dotenv

load_dotenv()
genai.configure(api_key=os.getenv("GOOGLE_API_KEY")) # type: ignore

print("📡 Asking Google for available models...")
for m in genai.list_models(): # type: ignore
    if 'generateContent' in m.supported_generation_methods:
        print(f"✅ Available: {m.name}")