#!bin/sh


docker build -t thincats-reports .
docker tag thincats-reports:latest asia.gcr.io/firm-champion-204312/thincats-reports:latest
docker push asia.gcr.io/firm-champion-204312/thincats-reports:latest
# runs on a micro without any overhead (not a gce docker micro as fluentd runs on that at a whopping 300mb) and should have lots of space on a small
# gcloud compute instances create-with-container reports --machine-type g1-small --zone australia-southeast1-a --container-image asia.gcr.io/firm-champion-204312/thincats-reports:latest
gcloud beta compute instances update-container reports
echo "To finish a deployment, dont forget to enable http and https traffic on the new instance AND attach the thincats permanent IP address to it"