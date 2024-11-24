#!/bin/bash

sudo yum update -y
sudo amazon-linux-extras install docker

sudo yum install -y git memcached
sudo usermod -a -G docker ec2-user

sudo systemctl start docker

sudo systemctl start memcached

sudo systemctl enable memcached

sudo yum install -y nginx certbot python3-certbot-nginx

sudo cp nginx.conf /etc/nginx
sudo certbot --nginx -d shrtn.cloud
