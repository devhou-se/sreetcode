from abc import ABC, abstractmethod
import logging
import requests
import time

import gen.sreeify_pb2 as sreeify_pb2
import gen.sreeify_pb2_grpc as sreeify_pb2_grpc
from sreeify import sreeify_text


class Replacement:
    def __init__(self, original, replacement):
        self.original = original
        self.replacement = replacement


class Request(ABC):
    @abstractmethod
    def do(self, link_replacements) -> str:
        raise NotImplementedError()

    @staticmethod
    def build(request) -> "Request":
        data_field = request.WhichOneof("data")
        if data_field == "payload":
            req = PayloadRequest(request.payload)
        elif data_field == "url":
            req = UrlRequest(request.url)
        else:
            raise ValueError("Invalid request")
        return req


class PayloadRequest(Request):
    def __init__(self, payload):
        self.payload = payload

    def do(self, link_replacements):
        text = sreeify_text(self.payload, link_replacements)
        return text


class UrlRequest(Request):
    def __init__(self, url):
        self.url = url

    def do(self, link_replacements):
        payload = requests.get(self.url).text
        text = sreeify_text(payload, link_replacements)
        return text


class SreeificationService(sreeify_pb2_grpc.SreeificationService):
    def Sreeify(self, request, context):
        start = time.time()

        req = Request.build(request)
        resp = req.do(request.link_replacements)

        response = sreeify_pb2.SreeifyResponse(payload=resp)
        end = time.time()
        logging.info(f"Request took {end - start} seconds")
        return response
