import os
import time
import random
import logging
import psycopg2

# Configure logging to match production standards
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)
logger = logging.getLogger("metrix-collector")

PLATFORMS = ["youtube", "instagram", "tiktok"]
DATABASE_URL = os.getenv("DATABASE_URL", "postgres://metrix:metrix_password@postgres:5432/metrix?sslmode=disable")

def get_db_connection():
    try:
        conn = psycopg2.connect(DATABASE_URL)
        return conn
    except Exception as e:
        logger.error(f"Failed to connect to database: {e}")
        return None

def find_target_user(conn):
    """Find the target user_id or return None if no user exists yet."""
    user_id_env = os.getenv("USER_ID")
    if user_id_env:
        return user_id_env
    
    try:
        with conn.cursor() as cur:
            cur.execute("SELECT id FROM users ORDER BY created_at ASC LIMIT 1;")
            row = cur.fetchone()
            if row:
                return row[0]
    except Exception as e:
        logger.error(f"Error querying users table: {e}")
        conn.rollback()
    return None

def simulate_and_persist(conn, user_id):
    """Generates simulated metric data for platforms and writes to PostgreSQL."""
    try:
        total_reach = 0
        total_engagement = 0.0
        total_growth = 0
        
        with conn.cursor() as cur:
            for platform in PLATFORMS:
                reach = random.randint(5000, 35000)
                engagement = round(random.uniform(2.5, 9.5), 2)
                growth = random.randint(50, 400)

                total_reach += reach
                total_engagement += engagement
                total_growth += growth

                # Insert timeseries entry for platform
                cur.execute("""
                    INSERT INTO metrics_timeseries (user_id, platform, metric_date, value)
                    VALUES (%s, %s, CURRENT_DATE, %s);
                """, (user_id, platform, reach))

            avg_engagement = round(total_engagement / len(PLATFORMS), 2)

            # Insert overall metrics summary entry
            cur.execute("""
                INSERT INTO metrics_summary (user_id, total_reach, avg_engagement, follower_growth, recorded_at)
                VALUES (%s, %s, %s, %s, NOW());
            """, (user_id, total_reach, avg_engagement, total_growth))

        conn.commit()
        logger.info(f"Ingested metrics for user {user_id}: total_reach={total_reach}, avg_engagement={avg_engagement}%, follower_growth={total_growth}")

    except Exception as e:
        conn.rollback()
        logger.error(f"Failed to persist metrics: {e}")

def main():
    logger.info("Metrix Data Pipeline Collector started.")
    
    while True:
        conn = get_db_connection()
        if conn:
            user_id = find_target_user(conn)
            if user_id:
                logger.info(f"Starting ingestion cycle for user_id={user_id}...")
                simulate_and_persist(conn, user_id)
            else:
                logger.warning("No user found in database yet. Waiting for a user to register...")
            conn.close()
        else:
            logger.warning("Database connection unavailable. Retrying in 10s...")

        logger.info("Cycle complete. Sleeping for 15 seconds...")
        time.sleep(15)

if __name__ == "__main__":
    main()
