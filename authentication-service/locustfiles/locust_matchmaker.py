# TORUN :
# time locust -f locustfiles/locust_matchmaker.py --host=localhost:8000 -c 1 -r 1 --no-web 2>&1 | tee 111.txt

import inspect
import time
import socket
import requests
from locust import Locust, TaskSet, task, events
from random import randint

MATCHMAKER_ADDRESS = "X.X.Y.Y"


def stopwatch(func):
    def wrapper(*args, **kwargs):
        # get task's function name
        previous_frame = inspect.currentframe().f_back
        _, _, task_name, _, _ = inspect.getframeinfo(previous_frame)

        start = time.time()
        result = None
        try:
            result = func(*args, **kwargs)
        except Exception as e:
            total = int((time.time() - start) * 1000)
            events.request_failure.fire(request_type="Agones GKE", name=task_name, response_time=total, exception=e)
        else:
            total = int((time.time() - start) * 1000)
            events.request_success.fire(request_type="Agones GKE", name=task_name, response_time=total, response_length=0)
        return result

    return wrapper


class AutoscalerClient:
    def __init__(self, host):
        host = host.split(':')
        self.host = str(host[0])
        self.port = int(host[1])

    @stopwatch
    def create_new_player(self, server_ip, server_port):
        udp_socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        udp_socket.connect((server_ip, int(server_port)))
        udp_socket.setblocking(0)
        udp_socket.send("NEW")
        time.sleep(0.15)
        udp_socket.close()
        return None

    def allocate_gameserver_ip_address(self, lat, lon):
        try:
            gs = requests.get('http://' + MATCHMAKER_ADDRESS + '/matchmaker/v1/allocate/lat/{}/lon/{}'.format(lat, lon), timeout=2, verify=False).json()
            if len(gs['status']['ports']) == 1:
                print ("The Cell ID is: " + str(gs['status']['cellId']))
                print ('http://' + MATCHMAKER_ADDRESS + '/matchmaker/v1/allocate/lat/{}/lon/{}'.format(lat, lon))
                return [gs['status']['address'], gs['status']['ports'][0]['port']]
            else:
                return []
        except Exception as e:
            print (" >>> Error contacting the game matcher <<< " + str(e))
            # time.sleep(20)
            return []


class AutoscalerLocust(Locust):
    def __init__(self):
        super(AutoscalerLocust, self).__init__()
        self.client = AutoscalerClient(self.host)


class AutoscalerTasks(TaskSet):

    @task
    def add_player(self):
        locations = [
            {"lat": "43.652690", "lon": "-79.373590"},
            {"lat": "43.545050", "lon": "-79.742600"},
            {"lat": "43.545360", "lon": "-79.742280"},
            {"lat": "49.285497", "lon": "-123.119908"},  # Burrad (Vancouver)
            {"lat": "49.286835", "lon": "-123.120956"},  # Bental (Vancouver)
            {"lat": "49.283504", "lon": "-123.119016"},  # W Georgia (Vancouver)
            {"lat": "45.504038", "lon": "-73.571641"},  # McGill (Montreal)
            {"lat": "45.504286", "lon": "-73.568734"},  # Place Phillips (Montreal)
            {"lat": "45.502271", "lon": "-73.571706"},  # McGill College Ave (Montreal)
            {"lat": "43.653370", "lon": "-79.369177"},  # Richmond St E (Toronto)
            {"lat": "43.651765", "lon": "-79.370360"},  # Adelaide St E (Toronto)
            {"lat": "43.651610", "lon": "-79.364153"},  # Berkeley St (Toronto)
            {"lat": "51.057395", "lon": "-114.035526"}  # Northeast (Calgary)
        ]
        counter = 0
        while counter < 100:
            ranint = randint(0, len(locations) - 1)
            match_address = self.client.allocate_gameserver_ip_address(locations[ranint]['lat'], locations[ranint]['lon'])
            if len(match_address) == 2:
                self.client.create_new_player(match_address[0], match_address[1])
                time.sleep(2)
                print ("Added Player on Server: " + str(match_address[0]) + ":" + str(match_address[1])) + " --> " + str(counter + 1)
                counter += 1
            else:
                print (" /// Error allocating a server from the matchmaker /// ")
        exit()


class AutoscalerUser(AutoscalerLocust):
    task_set = AutoscalerTasks
    min_wait = 0
    max_wait = 0
