FROM golang:1.7

ADD mapbot /mapbot
ADD run.sh /run.sh

ENTRYPOINT ["/run.sh"]
