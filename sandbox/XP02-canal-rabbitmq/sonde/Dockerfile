ARG BASE_IMAGE
FROM ${BASE_IMAGE}
COPY requirements.txt /sonde/requirements.txt
RUN pip install --no-cache-dir --require-hashes --only-binary=:all: -r /sonde/requirements.txt
COPY sonde.py /sonde/sonde.py
USER 65534:65534
ENTRYPOINT ["python", "-I", "/sonde/sonde.py"]
