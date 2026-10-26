# Image de la sonde du canal (outils/verifier.sh). Le script est monte a l'execution : l'image ne porte que pika.
FROM python:3.13.17-slim@sha256:d98cc29d3266e535dd9da1099688d9cb3ca4753813571e071763d17725460962
COPY requirements.txt /sonde/requirements.txt
RUN pip install --no-cache-dir --require-hashes --only-binary=:all: -r /sonde/requirements.txt
USER 65534:65534
ENTRYPOINT ["python", "-I"]
