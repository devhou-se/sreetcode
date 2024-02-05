import logging
import time

import gen.sreeify_pb2 as sreeify_pb2
import gen.sreeify_pb2_grpc as sreeify_pb2_grpc
from sreeify import sreeify_text

CHUNK_SIZE = 1024 * 1024  # 1MB
ENCODING = "utf-8"


class SreeificationService(sreeify_pb2_grpc.SreeificationService):
    def Sreeify(self, sreequest_iterator: list[sreeify_pb2.Sreequest], target):
        # Receive and process request parts
        req_id = None
        start = time.time()
        parts = []
        for sreequest in sreequest_iterator:
            if req_id is None:
                req_id = sreequest.id
            if len(parts) < sreequest.total_parts:
                parts += [None] * (sreequest.total_parts - len(parts))
            parts[sreequest.part] = sreequest.data
            if None in parts:
                continue
            break

        flat_bytes = b"".join(parts)
        payload = flat_bytes.decode(ENCODING)
        logging.info(f"Received request {req_id} with {len(parts)} parts and {len(flat_bytes)} bytes")

        # Process request
        resp = sreeify_text(payload)

        # Chunk and send response parts
        chunks = [resp[i:i + CHUNK_SIZE] for i in range(0, len(resp), CHUNK_SIZE)]
        for i, chunk in enumerate(chunks):
            yield sreeify_pb2.Sreesponse(
                id=req_id,
                part=i,
                total_parts=len(chunks),
                data=bytes(chunk, ENCODING)
            )

        logging.info(f"Sent response {req_id} with {len(chunks)} parts and {len(resp)} bytes")

        end = time.time()
        logging.info(f"Request took {end - start} seconds")
        return
