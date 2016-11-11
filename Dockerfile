FROM debian:stable

RUN for i in $(seq 1 5); do apt-get update && break; done; \
    for i in $(seq 1 5); do apt-get -y install ca-certificates && break; done

ADD mapbot /mapbot
ADD run.sh /run.sh

CMD ["/run.sh"]
