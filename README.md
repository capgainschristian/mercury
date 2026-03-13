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

