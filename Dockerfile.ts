FROM node:16

RUN npm install -g typescript
ENTRYPOINT [ "/usr/local/bin/tsc" ]
WORKDIR /work
