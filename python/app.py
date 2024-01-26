from concurrent import futures
import logging
import os

import grpc

import gen.sreeify_pb2_grpc as sreeify_pb2_grpc
from server import SreeificationService


def main():
    logging.basicConfig(level=logging.INFO)

    port = os.environ.get("PORT", 50051)

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    sreeify_pb2_grpc.add_SreeificationServiceServicer_to_server(
        SreeificationService(), server
    )

    server.add_insecure_port(f"[::]:{port}")

    logging.info(f"Starting server on port {port}")
    server.start()

    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        server.stop(0)
        logging.info("Server stopped")


if __name__ == '__main__':
    main()
