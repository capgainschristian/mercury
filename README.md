Mercury

Bring up the entire stack:

```
docker compose up --build -d
```

If you want to use the Go container to develop and/or troubleshoot:

```
docker compose exec web sh
```

Teardown:

```
docker compose down
docker compose down -v (clean slate)
```

How to verify on Redis:

```
docker exec -it mercury-cache-1 redis-cli -a $REDIS_PASSWORD XRANGE platform-events - +
```

Other useful Redis commands:

```
XLEN platform-events

XINFO STREAM platform-events

XREVRANGE platform-events + - COUNT 10
```