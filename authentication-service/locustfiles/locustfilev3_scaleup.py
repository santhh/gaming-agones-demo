# TORUN: time locust -f locustfilev3_scaleup.py --host=localhost:8000 -c 1 -r 1 --no-web

import inspect
import time
import socket
import requests
import subprocess
import re
from locust import Locust, TaskSet, task, events

AUTH_USER = ""
AUTH_KEY = ""


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


# -------------------------------------------------------------------------


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

    def allocate_gameserver_ip_address(self):
        gs = requests.get('https://35.190.19.82/address', auth=(AUTH_USER, AUTH_KEY), verify=False).json()
        if len(gs['status']['ports']) == 1 and gs['status']['state'] == "Allocated":
            return [gs['status']['address'], gs['status']['ports'][0]['port']]
        else:
            return []

    def list_ready_gameservers(self):
        all_gs = subprocess.check_output(["kubectl", "get", "gs", "-o=custom-columns=NAME:.metadata.name,STATUS:.status.state,IP:.status.address,PORT:.status.ports"])
        ready = []
        for gs in all_gs.splitlines():
            if "Ready" in gs:
                ip = re.findall(r'[0-9]+(?:\.[0-9]+){3}', gs)
                port = re.findall(r':[0-9]{4}', gs)
                if ip and port:
                    ready.append([ip[0], port[0][1:]])
        return ready

    def list_allocated_gameservers(self):
        all_gs = subprocess.check_output(["kubectl", "get", "gs", "-o=custom-columns=NAME:.metadata.name,STATUS:.status.state,IP:.status.address,PORT:.status.ports"])
        allocated = []
        for gs in all_gs.splitlines():
            if "Allocated" in gs:
                ip = re.findall(r'[0-9]+(?:\.[0-9]+){3}', gs)
                port = re.findall(r':[0-9]{4}', gs)
                if ip and port:
                    allocated.append([ip[0], port[0][1:]])
        return allocated


# -------------------------------------------------------------------------


class AutoscalerLocust(Locust):
    def __init__(self):
        super(AutoscalerLocust, self).__init__()
        self.client = AutoscalerClient(self.host)


class AutoscalerTasks(TaskSet):

    @task
    def add_player(self):
        num_alloc_gs = len(self.client.list_allocated_gameservers())
        num_ready_gs = len(self.client.list_ready_gameservers())

        if (num_alloc_gs) < 10:
            if (num_ready_gs == 0):
                print ("Waiting for new GameServers to get 'Ready' [Waiting 10 Seconds]")
                time.sleep(10)
            else:
                allocated_gs = self.client.allocate_gameserver_ip_address()
                if len(allocated_gs) == 2:
                    server_ip = allocated_gs[0]
                    server_port = allocated_gs[1]
                    print ("Allocated Gameserver: " + str(server_ip) + ":" + str(server_port))
                    time.sleep(20)

                    for i in range(0, 10):
                        self.client.create_new_player(server_ip, server_port)
                        time.sleep(2.5)
                        print ("Added Player '" + str(i + 1) + "' on Server: " + str(server_ip) + ":" + str(server_port))
                else:
                    print ("There was error Allocating a GS: No JSON object could be decoded")
        else:
            print ("Total number of 'Allocated' server reached the limit... Goodbye!")
            exit()


class AutoscalerUser(AutoscalerLocust):
    task_set = AutoscalerTasks
    min_wait = 0
    max_wait = 0
