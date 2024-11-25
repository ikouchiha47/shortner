#!/bin/bash

# sudo mkdir -p /app
# sudo chown -R ec2-user:ec2-user /app

sudo amazon-linux-extras enable epel
sudo yum clean metadata; sudo yum -y install epel-release; sudo yum update -y

sudo yum install -y git memcached go nginx certbot certbot-nginx

pip3 install -U supervisor 

mkdir -p /tmp/{log,run}

sudo systemctl start memcached

cd app && \
	git clone https://github.com/ikouchiha47/shortner.git && \
	cd shortner && \
	go env -w GOPATH=/app/go; go env -w GOMODCACHE=/app/go/pkg/mod && \
	make build.all && \
	sudo cp supervisor.conf /etc/supervisord.conf 


sudo supervisord -c /etc/supervisord.conf

# sudo certbot --nginx -d shrtn.cloud
# sudo systemctl status certbot.timer

