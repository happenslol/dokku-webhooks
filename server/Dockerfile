FROM alpine

RUN mkdir /app && mkdir /app/storage
ADD . /app
WORKDIR /app
CMD [ "/app/server" ]
