import os

db = os.getenv("DATABASE_URL")
redis = os.environ["REDIS_URL"]
api = os.environ.get("MISSING_PY")
