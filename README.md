Mercury

Stand up Redis locally and check to make sure it's running:

docker run -d --name notification-redis -p 6379:6379 redis:latest

# Should return PONG

docker exec -it notification-redis redis-cli ping
