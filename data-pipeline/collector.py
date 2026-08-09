import os
import time
import random
import logging
import psycopg2

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)
logger = logging.getLogger("metrix-collector")

DATABASE_URL = os.getenv(
    "DATABASE_URL",
    "postgres://metrix:metrix_password@postgres:5432/metrix?sslmode=disable"
)

CONTENT_TITLE_POOL = {
    "youtube": {
        "titles": [
            "How to Grow on YT in 2026",
            "Behind the Scenes: My Filming Setup",
            "3 Mistakes Killing Your CTR",
            "I Tested 1000 Creators' Hooks",
            "Reacting to My First Video Ever",
            "The One Metric That Matters",
            "Why My Views Dropped Overnight",
        ],
        "base_reach": 12000,
        "reach_variance": 25000,
    },
    "instagram": {
        "titles": [
            "Day in the Life of a Creator",
            "Carousel: 10 Reel Ideas That Work",
            "Golden Hour Photo Dump",
            "Story: Q&A - Your Questions Answered",
            "Reel: One Year of Consistency",
            "The Algorithm Secret Explained",
            "Behind Every Viral Post",
        ],
        "base_reach": 8000,
        "reach_variance": 18000,
    },
    "tiktok": {
        "titles": [
            "Metrix Alpha Reveal!",
            "POV: The Algorithm Finally Found You",
            "Trending Sound Challenge",
            "How I Edit in Under 10 Minutes",
            "Ghosting the Haters: Week 3",
            "60 Seconds That Broke My Channel",
            "The Hook That Got Me 1M Views",
        ],
        "base_reach": 15000,
        "reach_variance": 35000,
    },
}

AGE_BRACKETS = ["13-17", "18-24", "25-34", "35-44", "45-54", "55+"]
GENDER_LABELS = ["Male", "Female"]
COUNTRY_LABELS = ["USA", "UK", "Canada", "Australia", "Germany", "Brazil", "India"]


def get_db_connection():
    try:
        conn = psycopg2.connect(DATABASE_URL)
        return conn
    except Exception as e:
        logger.error(f"Failed to connect to database: {e}")
        return None


def find_target_user(conn):
    if conn is None:
        return []
    user_id_env = os.getenv("USER_ID")
    if user_id_env:
        return [user_id_env]
    try:
        with conn.cursor() as cur:
            cur.execute("SELECT id FROM users ORDER BY created_at ASC;")
            rows = cur.fetchall()
            if rows:
                return [row[0] for row in rows]
    except Exception as e:
        logger.error(f"Error querying users table: {e}")
        conn.rollback()
    return []


def get_connected_platforms(conn, user_id):
    if conn is None:
        return []
    try:
        with conn.cursor() as cur:
            cur.execute("""
                SELECT platform FROM platform_accounts
                WHERE user_id = %s AND status = 'connected'
                ORDER BY created_at ASC;
            """, (user_id,))
            return [row[0] for row in cur.fetchall()]
    except Exception as e:
        logger.error(f"Error querying platform accounts: {e}")
        conn.rollback()
        return []


def insert_timeseries(conn, user_id, platform, reach):
    try:
        with conn.cursor() as cur:
            cur.execute("""
                INSERT INTO metrics_timeseries (user_id, platform, metric_date, value)
                VALUES (%s, %s, CURRENT_DATE, %s)
                ON CONFLICT (user_id, platform, metric_date)
                DO UPDATE SET value = EXCLUDED.value
            """, (user_id, platform, reach))
    except Exception as e:
        logger.error(f"Failed to insert timeseries for {platform}: {e}")
        conn.rollback()


def insert_content_items(conn, user_id, platform, pool):
    titles = random.sample(pool["titles"], min(3, len(pool["titles"])))
    for title in titles:
        half_var = pool["reach_variance"] // 2
        reach = pool["base_reach"] + random.randint(-half_var, pool["reach_variance"])
        engagement = round(random.uniform(2.5, 12.0), 2)
        try:
            with conn.cursor() as cur:
                cur.execute(
                    "INSERT INTO content_items "
                    "(user_id, platform, title, engagement, reach, recorded_at) "
                    "VALUES (%s, %s, %s, %s, %s, CURRENT_DATE)",
                    (user_id, platform, title, engagement, reach),
                )
        except Exception as e:
            logger.error(f"Failed to insert content item '{title}': {e}")
            conn.rollback()


def upsert_audience_insight(conn, user_id, category, label, value):
    try:
        with conn.cursor() as cur:
            cur.execute("""
                INSERT INTO audience_insights (user_id, category, label, value, recorded_at)
                VALUES (%s, %s, %s, %s, CURRENT_DATE)
                ON CONFLICT (user_id, category, label, DATE(recorded_at))
                DO UPDATE SET value = EXCLUDED.value
            """, (user_id, category, label, value))
    except Exception as e:
        logger.error(f"Failed to upsert audience insight {category}/{label}: {e}")
        conn.rollback()


def generate_audience_data(conn, user_id, platforms):
    if not platforms:
        return

    total_weight = len(platforms)
    platform_weights = {}
    for p in platforms:
        platform_weights[p] = 1.0 / total_weight

    for age in AGE_BRACKETS:
        if age in ("13-17", "55+"):
            pct = round(random.uniform(3.0, 8.0), 1)
        elif age in ("45-54",):
            pct = round(random.uniform(6.0, 12.0), 1)
        elif age == "18-24":
            pct = round(random.uniform(20.0, 35.0), 1)
        elif age == "25-34":
            pct = round(random.uniform(25.0, 40.0), 1)
        else:
            pct = round(random.uniform(10.0, 20.0), 1)
        upsert_audience_insight(conn, user_id, "age", age, pct)

    male = round(random.uniform(48.0, 55.0), 1)
    female = round(100.0 - male, 1)
    upsert_audience_insight(conn, user_id, "gender", "Male", male)
    upsert_audience_insight(conn, user_id, "gender", "Female", female)

    country_values = {c: round(random.uniform(8.0, 22.0), 1) for c in COUNTRY_LABELS}
    total = sum(country_values.values())
    for c, v in country_values.items():
        normalized = round((v / total) * 100, 1)
        upsert_audience_insight(conn, user_id, "country", c, normalized)


def simulate_and_persist(conn, user_id):
    platforms = get_connected_platforms(conn, user_id)
    if not platforms:
        logger.warning(f"No connected platforms for user {user_id}; skipping ingestion.")
        return

    try:
        total_reach = 0
        total_engagement = 0.0
        total_growth = 0

        for platform in platforms:
            if platform not in CONTENT_TITLE_POOL:
                continue
            pool = CONTENT_TITLE_POOL[platform]
            half_var = pool["reach_variance"] // 2
            reach = pool["base_reach"] + random.randint(-half_var, pool["reach_variance"])
            engagement = round(random.uniform(2.5, 9.5), 2)
            growth = random.randint(50, 400)

            total_reach += reach
            total_engagement += engagement
            total_growth += growth

            insert_timeseries(conn, user_id, platform, reach)
            insert_content_items(conn, user_id, platform, pool)

        avg_engagement = round(total_engagement / len(platforms), 2)

        with conn.cursor() as cur:
            cur.execute(
                "INSERT INTO metrics_summary "
                "(user_id, total_reach, avg_engagement, follower_growth, recorded_at) "
                "VALUES (%s, %s, %s, %s, NOW())",
                (user_id, total_reach, avg_engagement, total_growth),
            )

        generate_audience_data(conn, user_id, platforms)

        conn.commit()
        logger.info(
            f"Ingested metrics for user {user_id}: "
            f"total_reach={total_reach}, avg_engagement={avg_engagement}%, "
            f"follower_growth={total_growth}, platforms={len(platforms)}"
        )

    except Exception as e:
        conn.rollback()
        logger.error(f"Failed to persist metrics: {e}")


def main():
    logger.info("Metrix Data Pipeline Collector started.")

    while True:
        conn = get_db_connection()
        if conn:
            user_ids = find_target_user(conn)
            if user_ids:
                for user_id in user_ids:
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
