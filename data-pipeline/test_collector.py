import os
from collector import (
    get_db_connection,
    find_target_user,
    get_connected_platforms,
    CONTENT_TITLE_POOL,
    AGE_BRACKETS,
    GENDER_LABELS,
    COUNTRY_LABELS,
)


def test_get_db_connection_returns_none_on_bad_url():
    os.environ["DATABASE_URL"] = "postgresql://bad:bad@badhost:5432/bad"
    conn = get_db_connection()
    assert conn is None


def test_get_connected_platforms_handles_none_conn():
    platforms = get_connected_platforms(None, "test-user")
    assert platforms == []


def test_find_target_user_handles_none_conn():
    user_ids = find_target_user(None)
    assert user_ids == []


def test_content_title_pool_has_expected_platforms():
    assert "youtube" in CONTENT_TITLE_POOL
    assert "instagram" in CONTENT_TITLE_POOL
    assert "tiktok" in CONTENT_TITLE_POOL
    for platform, pool in CONTENT_TITLE_POOL.items():
        assert len(pool["titles"]) > 0
        assert pool["base_reach"] > 0


def test_age_brackets_and_countries_exist():
    assert len(AGE_BRACKETS) > 0
    assert len(GENDER_LABELS) > 0
    assert len(COUNTRY_LABELS) > 0


def test_content_title_pool_variance_is_positive():
    for platform, pool in CONTENT_TITLE_POOL.items():
        assert pool["reach_variance"] > 0
