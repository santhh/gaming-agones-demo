# To run -> locust -f locustfile.py --host=http://35.226.10.79:80

from locust import HttpLocust, TaskSet, task
import json
import time


payload = {
    "email": "",
    "password": ""
}
token = ""
ref_token = ""


class MyTaskSet(TaskSet):

    def on_start(self):
        res = self.client.post("/loginplayer", json=payload, name="Logging In")
        json_data = json.loads(res.text)
        global token, ref_token
        token = json_data['idToken']
        ref_token = json_data['refToken']
        res.raise_for_status()
        time.sleep(10)

    def on_stop(self):
        res = self.client.put(
            "/logout/" + payload['email'], name="Logging Out")
        res.raise_for_status()

    @task
    def updateactivity(self):
        res = self.client.get("/updateactivity/" + payload['email'], name="Updating Activity")
        res.raise_for_status()

    @task
    def getplayerinfo(self):
        res = self.client.get("/getplayerinfo/" + token, name="Getting player info")
        res.raise_for_status()

    @task
    def authenticationtokenrefresh(self):
        res = self.client.get(
            "/authenticationtokenrefresh/" + ref_token, name="Refreshing token")
        res.raise_for_status()


class MyLocust(HttpLocust):
    task_set = MyTaskSet
    min_wait = 4000
    max_wait = 9000
