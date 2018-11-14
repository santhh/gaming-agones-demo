# TORUN: time locust -f locustfilev3_scaledown.py --host=localhost:8000 -c 1 -r 1 --no-web

import inspect
import time
import socket
import subprocess
import re

from locust import Locust, TaskSet, task, events


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
    def de_allocate_gameserver(self, server_ip, server_port):
        udp_socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        udp_socket.connect((server_ip, int(server_port)))
        udp_socket.setblocking(0)
        udp_socket.send("EXIT")
        time.sleep(0.15)
        return None

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
    def remove_player(self):
        allocated_gs = self.client.list_allocated_gameservers()

        if len(allocated_gs) == 0:
            print ("Test Ended... No more 'Allocated' servers")
            exit()
        else:
            for server in allocated_gs:
                server_ip = server[0]
                server_port = server[1]
                self.client.de_allocate_gameserver(server_ip, server_port)
                print ("De-allocated Game Server >>>>> " + str(server_ip) + ":" + str(server_port))
                time.sleep(5)


class AutoscalerUser(AutoscalerLocust):
    task_set = AutoscalerTasks
    min_wait = 0
    max_wait = 0
