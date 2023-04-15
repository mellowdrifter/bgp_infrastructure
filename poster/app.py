#!/usr/bin/env python3

import tweepy
import os
import logging
import grpc
import tweeter_pb2 as pb
import tweeter_pb2_grpc
from typing import Text
from concurrent import futures

_PORT = os.environ["PORT"]


class Poster(tweeter_pb2_grpc.PosterServicer):
    def PostMessages(self, request, context):
        logging.info("Request received to post a message")
        return post_messages(request)


def post_messages(request: pb.PostMessageRequest()) -> pb.PostMessageResponse():
    logging.info("got the following request: " + request)
    msg = request.message
    consumer_key = request.twitter_account.consumer_key
    consumer_secret = request.twitter_account.consumer_secret
    access_token = request.twitter_account.access_token
    access_secret = request.twitter_account.access_secret

    auth = tweepy.OAuthHandler(consumer_key, consumer_secret)
    auth.set_access_token(access_token, access_secret)
    api = tweepy.API(auth)
    media = api.media_upload("test.png")

    tweet = "test with image"
    status = api.update_status(status=tweet, media_ids=[media.media_id])
    return pb.PostMessageResponse(response=status)


def _serve(port: Text):
    bind_address = f"[::]:{port}"
    server = grpc.server(futures.ThreadPoolExecutor())
    tweeter_pb2_grpc.add_GrapherServicer_to_server(Poster(), server)
    server.add_insecure_port(bind_address)
    server.start()
    logging.info("Listening on %s", bind_address)
    server.wait_for_termination()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    _serve(_PORT)
