FROM python:3.13.3-slim-bookworm AS build-env

ADD ./ ./

RUN pip3 install --upgrade pip && \
    pip install -r ./requirements.txt

CMD [ "python3", "./app.py"]