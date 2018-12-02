# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
# build with
# docker build -t thincats-reports .
# docker tag thincats-reports:latest asia.gcr.io/firm-champion-204312/thincats-reports:latest
# docker push asia.gcr.io/firm-champion-204312/thincats-reports:latest
# gcloud compute instances create-with-container reports --machine-type f1-micro --zone australia-southeast1-a --container-image asia.gcr.io/firm-champion-204312/thincats-reports:latest
FROM golang AS build-env

COPY . ./src/github.com/dolmant/thincats-reports
RUN cd ./src/github.com/dolmant/thincats-reports && go get
RUN CGO_ENABLED=0 go build -o ./src/github.com/dolmant/thincats-reports/thincats-reports ./src/github.com/dolmant/thincats-reports/

FROM alpine
ADD ca-certificates.crt /etc/ssl/certs/
WORKDIR /
RUN mkdir /root/thincats-reports
COPY --from=build-env /go/src/github.com/dolmant/thincats-reports /root/thincats-reports
ENTRYPOINT /root/thincats-reports/thincats-reports

EXPOSE 8079
