import json
import time
import random
import logging

# Configure logging to match the "Production Traceability" standards
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)
logger = logging.getLogger("metrix-collector")

PLATFORMS = ["youtube", "instagram", "tiktok"]
METRICS = ["reach", "engagement_rate", "follower_count"]

def simulate_fetch(platform):
    """Simulates fetching data from a platform API."""
    logger.info(f"Connecting to {platform.upper()} OAuth gateway...")
    time.sleep(0.5) # Simulate network latency
    
    data = {
        "platform": platform,
        "metrics": {
            "reach": random.randint(1000, 50000),
            "engagement_rate": round(random.uniform(1.5, 12.0), 2),
            "follower_count": random.randint(500, 100000)
        },
        "timestamp": int(time.time()),
        "status": "success"
    }
    return data

def main():
    logger.info("Metrix Data Pipeline Collector started.")
    logger.info("Initializing multi-tenant ingestion cycle...")

    try:
        while True:
            for platform in PLATFORMS:
                data = simulate_fetch(platform)
                logger.info(f"Ingested {platform.upper()} metrics: {json.dumps(data['metrics'])}")
            
            logger.info("Cycle complete. Sleeping for 60 seconds...")
            time.sleep(60)
    except KeyboardInterrupt:
        logger.info("Collector shutting down gracefully.")

if __name__ == "__main__":
    main()
