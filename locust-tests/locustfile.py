from locust import HttpUser, task, between
from locust.env import Environment
import os

class NginxUser(HttpUser):
    wait_time = between(1, 5)  # Simulates wait time between requests

    # Read the target address from an environment variable or use a default

    @task
    def get_homepage(self):
        # Define the task that hits the root of the NGINX server
        self.client.get("/")


# This section allows running the test outside of Locust's web UI if needed
if __name__ == "__main__":
    import locust

    env = Environment(user_classes=[NginxUser])
    env.create_local_runner()
    env.runner.start(1, spawn_rate=1)  # Start with 1 user and 1 user per second spawn rate
    locust.stats_printer(env.stats)  # Print stats in real-time
    env.runner.greenlet.join()  # Keep the runner running
