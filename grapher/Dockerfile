FROM python:3.12.4-bookworm  AS build-env

ADD ./ ./

RUN pip3 install --upgrade pip && \
    pip install -r ./requirements.txt

CMD [ "python3", "./app.py"]