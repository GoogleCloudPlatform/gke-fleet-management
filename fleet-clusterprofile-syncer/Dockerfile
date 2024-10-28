FROM google/cloud-sdk:latest

RUN apt-get update && apt-get install -qqy kubectl uuid-runtime jq

ADD membership-to-clusterprofile.sh /usr/sbin/membership-to-clusterprofile.sh
