import logging

import gen.sreeify_pb2 as sreeify_pb2
import gen.sreeify_pb2_grpc as sreeify_pb2_grpc
from sreeify import sreeify


class Request:
    def do(self):
        raise NotImplementedError()


class PayloadRequest(Request):
    def __init__(self, payload):
        self.payload = payload

    def do(self):
        text = sreeify(self.payload)
        return text


class UrlRequest(Request):
    def __init__(self, url):
        self.url = url

    def do(self):
        return self.url


class SreeificationService(sreeify_pb2_grpc.SreeificationService):
    def Sreeify(self, request, context):
        data_field = request.WhichOneof("data")
        if data_field == "payload":
            req = PayloadRequest(request.payload)
        elif data_field == "url":
            req = UrlRequest(request.url)
        else:
            raise ValueError("Invalid request")

        resp = req.do()

        response = sreeify_pb2.SreeifyResponse(payload=resp)
        return response
