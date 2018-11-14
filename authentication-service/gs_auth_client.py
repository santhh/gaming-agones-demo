# TODO (Patrick): (COPYRIGHT NOTICE)

import grpc
import gs_auth_pb2
import gs_auth_pb2_grpc
import time
import argparse


def run(host):
    with grpc.insecure_channel(host) as channel:
        stub = gs_auth_pb2_grpc.GameserverAuthenticationStub(channel)

        email = raw_input("Enter your email: ")
        password = raw_input("Enter your password: ")

        print("----------------------------------- Create Player ------------------------------------------")
        try:
            register = stub.RegisterPlayer(
                gs_auth_pb2.PlayerRegistrationRequest(email=email, password=password))
            print(register)
        except grpc.RpcError as e:
            status_code = e.code()
            print ("Status Code: " + str(status_code.value))
        except:
            print ("Something wrong happened in RegisterPlayer()")

        print("---------------------------------------- Login Player ---------------------------------------")
        time.sleep(3)
        try:
            result = stub.LoginPlayer(
                gs_auth_pb2.LoginRequest(email=email, password=password))
            print(result)
        except grpc.RpcError as e:
            status_code = e.code()
            print ("Status Code: " + str(status_code.value))
        except:
            print ("Something wrong happened in LoginPlayer()")

        print("--------------------------------------- Get Player Info --------------------------------------")
        time.sleep(3)
        try:
            result.idToken
            info = stub.GetPlayerInfo(
                gs_auth_pb2.PlayerInfoRequest(idToken=result.idToken))
            print(info)
        except grpc.RpcError as e:
            status_code = e.code()
            print ("Status Code: " + str(status_code.value))
        except:
            print ("Something wrong happened in GetPlayerInfo()")

        print("----------------------------------------- Refresh Token --------------------------------------")
        time.sleep(3)
        try:
            token = stub.AuthenticationTokenRefresh(
                gs_auth_pb2.AuthenticationTokenRefreshRequest(idToken=result.refToken))
            print(token)
        except grpc.RpcError as e:
            status_code = e.code()
            print ("Status Code: " + str(status_code.value))
        except:
            print ("Something wrong happened in AuthenticationTokenRefresh()")

        print("-------------------------------------- Update Presence --------------------------------------")
        time.sleep(3)
        try:
            token = stub.UpdateActivity(
                gs_auth_pb2.UpdateActivityRequest(email=email))
            print(token)
        except grpc.RpcError as e:
            status_code = e.code()
            print ("Status Code: " + str(status_code.value))
        except:
            print ("Something wrong happened in UpdateActivity()")

        print("--------------------------------------- Logout Player ---------------------------------------")
        time.sleep(3)
        try:
            stub.LogoutPlayer(gs_auth_pb2.LogoutRequest(email=email))
        except grpc.RpcError as e:
            status_code = e.code()
            print ("Status Code: " + str(status_code.value))
        except:
            print ("Something wrong happened in LogoutPlayer()")


if __name__ == '__main__':
    # Parses the command-line supplied parameters
    parser = argparse.ArgumentParser()
    parser.add_argument('--host', default='localhost:50051', help='The server host.')
    args = parser.parse_args()
    run(args.host)
