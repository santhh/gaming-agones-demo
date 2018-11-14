# TODO (Patrick): (COPYRIGHT NOTICE)

from concurrent import futures
import time
import grpc
import gs_auth_pb2
import gs_auth_pb2_grpc
import pyrebase
import argparse
import re

# Parses the command-line supplied parameters
parser = argparse.ArgumentParser()
parser.add_argument("--apikey", required=True,
                    help="Enter your Firebase API key")
parser.add_argument("--authDomain", required=True,
                    help="Enter your Firebase AuthDomain")
parser.add_argument("--databaseURL", required=True,
                    help="Enter your Firebase Database URL")
parser.add_argument("--projectId", required=True,
                    help="Enter your Firebase Project ID")
parser.add_argument("--storageBucket", required=True,
                    help="Enter your Firebase Storage Bucket")
parser.add_argument("--messagingSenderId", required=True,
                    help="Enter your Firebase Messaging Sender ID")
args = parser.parse_args()

_ONE_DAY_IN_SECONDS = 60 * 60 * 24


# Initialize Firebase connection with the supplied command-line arguments
def contact_firebase():
    config = {
        "apiKey": args.apikey,
        "authDomain": args.authDomain,
        "databaseURL": args.databaseURL,
        "projectId": args.projectId,
        "storageBucket": args.storageBucket,
        "messagingSenderId": args.messagingSenderId
    }
    return pyrebase.initialize_app(config)


# TODO (Patrick): check if there is a better way to save emails in
# firebase database
def create_email(email_input):
    email_pattern = re.compile('([\w\-\.]+@(?:\w[\w\-]+\.)+[\w\-]+)')
    match = email_pattern.findall(email_input)
    if match:
        part1 = ''.join(match).split("@")
        part2 = part1[1].split(".")
        email_output = part1[0] + "-at-" + part2[0] + "-dot-" + part2[1]
        return email_output
    else:
        return False


class GameserverAuthenticationServicer(gs_auth_pb2_grpc.GameserverAuthenticationServicer):

    # Adds player in Firebase Authentication and Firebase Database
    def RegisterPlayer(self, request, context):
        firebase = contact_firebase()
        auth = firebase.auth()
        email_formatted = create_email(request.email)
        if email_formatted == False or len(request.password) < 6:
            context.set_details("Invalid Username or Password!!")
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
        else:
            register = auth.create_user_with_email_and_password(
                request.email, request.password)
            db = firebase.database()
            db.child("users").child(email_formatted).update(
                {"loggedIn": False, "loginTime": 0.0, "logoutTime": 0.0, "lastActivity": 0.0})
            return gs_auth_pb2.PlayerRegistrationResponse()

    # Login the player and sets "loggedIn=True" in Firebase Database
    def LoginPlayer(self, request, context):
        firebase = contact_firebase()
        auth = firebase.auth()
        timestamp = time.time()
        user = auth.sign_in_with_email_and_password(
            request.email, request.password)
        if user['idToken']:
            email_formatted = create_email(user['email'])
            db = firebase.database()
            db.child("users").child(email_formatted).update(
                {"loggedIn": True, "loginTime": timestamp, "lastActivity": timestamp})
            return gs_auth_pb2.LoginResponse(idToken=user['idToken'], email=user['email'],
                                             refToken=user['refreshToken'], loggedIn=True, time=timestamp)

    # Logout the player and sets the "loggedIn=False"
    def LogoutPlayer(self, request, context):
        firebase = contact_firebase()
        timestamp = time.time()
        email_formatted = create_email(request.email)
        db = firebase.database()
        player_exists_check = db.child("users").child(email_formatted).get()
        if player_exists_check.val():
            db.child("users").child(email_formatted).update(
                {"loggedIn": False, "logoutTime": timestamp})
            return gs_auth_pb2.LogoutResponse()
        else:
            context.set_details("Player not found!")
            context.set_code(grpc.StatusCode.NOT_FOUND)

    # Retrieves the player information from Firebase
    # TODO (Patrick): check if we need to keep the idToken or we need the
    # email from the API perspective
    def GetPlayerInfo(self, request, context):
        firebase = contact_firebase()
        auth = firebase.auth()
        try:
            information = auth.get_account_info(request.idToken)
        except:
            context.set_details("Invalid token!")
            context.set_code(grpc.StatusCode.PERMISSION_DENIED)
        if len(information['users']) == 1:
            db = firebase.database()
            userinfo = db.child("users").child(
                create_email(information['users'][0]['email'])).get()
            return gs_auth_pb2.PlayerInfoResponse(email=information['users'][0]['email'],
                                                  passwordUpdatedAt=str(information['users'][0]['passwordUpdatedAt']),
                                                  emailVerified=str(information['users'][0]['emailVerified']),
                                                  presence=userinfo.val()['loggedIn'],
                                                  lastActivity=userinfo.val()['lastActivity'])
        elif len(information['users']) == 0:
            context.set_details("Player information not found!")
            context.set_code(grpc.StatusCode.NOT_FOUND)
        elif len(information['users']) > 1:
            context.set_details("More than one record found!")
            context.set_code(grpc.StatusCode.OUT_OF_RANGE)

    # Refreshes the authentication token to extend the session
    def AuthenticationTokenRefresh(self, request, context):
        firebase = contact_firebase()
        auth = firebase.auth()
        token = auth.refresh(request.idToken)
        return gs_auth_pb2.AuthenticationTokenRefreshResponse(idToken=token['idToken'])

    # Updates the user's lastActivity in Firebase
    def UpdateActivity(self, request, context):
        firebase = contact_firebase()
        timestamp = time.time()
        email_formatted = create_email(request.email)
        db = firebase.database()
        player_exists_check = db.child("users").child(email_formatted).get()
        if player_exists_check.val():
            db = firebase.database()
            db.child("users").child(email_formatted).update(
                {"lastActivity": timestamp})
            return gs_auth_pb2.UpdateActivityResponse()
        else:
            context.set_details("Player not found!")
            context.set_code(grpc.StatusCode.NOT_FOUND)


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    gs_auth_pb2_grpc.add_GameserverAuthenticationServicer_to_server(
        GameserverAuthenticationServicer(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    try:
        while True:
            time.sleep(_ONE_DAY_IN_SECONDS)
    except KeyboardInterrupt:
        server.stop(0)


if __name__ == '__main__':
    serve()
