import logging
import time

import gen.sreeify_pb2 as sreeify_pb2
import gen.sreeify_pb2_grpc as sreeify_pb2_grpc
from sreeify import sreeify_text

CHUNK_SIZE = 1024 * 1024  # 1MB
ENCODING = "utf-8"


class SreeificationService(sreeify_pb2_grpc.SreeificationService):
    def Sreeify(self, sreequest_iterator: list[sreeify_pb2.Sreequest], target, *args, **kwargs):
        data: dict[str, list[bytes | None]] = {}

        for sreequest in sreequest_iterator:
            if sreequest.WhichOneof("data") == "ping":
                yield sreeify_pb2.Sreesponse(ping=sreequest.ping)
                continue

            payload = sreequest.payload
            if payload.id not in data:
                data[payload.id] = [None] * payload.total_parts

            data[payload.id][payload.part] = payload.data

            if all([datum is not None for datum in data[payload.id]]):
                flat_bytes = b"".join(data[payload.id])
                resp_data = flat_bytes.decode(ENCODING)
                logging.info(f"Received request {payload.id} with {len(data[payload.id])} parts and {len(flat_bytes)} bytes")
                resp = sreeify_text(resp_data)
                chunks = [resp[i:i + CHUNK_SIZE] for i in range(0, len(resp), CHUNK_SIZE)]
                for i, chunk in enumerate(chunks):
                    yield sreeify_pb2.Sreesponse(
                        payload=sreeify_pb2.Payload(
                            id=payload.id,
                            part=i,
                            total_parts=len(chunks),
                            data=bytes(chunk, ENCODING)
                        )
                    )
                logging.info(f"Sent response {payload.id} with {len(chunks)} parts and {len(resp)} bytes")
                del data[payload.id]
