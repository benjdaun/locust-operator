# Use the official Python image as the base
FROM python:3.9-slim

# Set working directory
WORKDIR /app

# Install Locust
RUN pip install --no-cache-dir locust

# Copy Locust test file to the container
COPY locustfile.py /app/

# Set default environment variables (which can be overridden)
ENV TARGET_URL=http://localhost:8080
ENV LOCUST_USERS=10
ENV LOCUST_SPAWN_RATE=2
ENV LOCUST_RUN_TIME=5m

# Default command to run Locust in headless mode, using env vars
CMD ["sh", "-c", "locust -f locustfile.py --headless -u $LOCUST_USERS -r $LOCUST_SPAWN_RATE --run-time $LOCUST_RUN_TIME --host $TARGET_URL"]
