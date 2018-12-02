#!bin/sh


docker build -t thincats-reports .
docker tag thincats-reports:latest asia.gcr.io/firm-champion-204312/thincats-reports:latest
docker push asia.gcr.io/firm-champion-204312/thincats-reports:latest
gcloud compute instances create-with-container reports  --container-image asia.gcr.io/firm-champion-204312/thincats-reports:latest
echo "To finish a deployment, dont forget to enable http and https traffic on the new instance AND attach the thincats permanent IP address to it"