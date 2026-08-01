import json
import time

def main():
    fake_metric = {
        "platform": "youtube",
        "metric": "subscribers",
        "value": 1250,
        "timestamp": int(time.time())
    }
    print(f"[DATA PIPELINE] Generated metric: {json.dumps(fake_metric)}")

if __name__ == "__main__":
    main()