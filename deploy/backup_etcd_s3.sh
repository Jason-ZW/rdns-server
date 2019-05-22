#!/bin/bash

# https://github.com/giantswarm/etcd-backup

export ETCDBACKUP_AWS_ACCESS_KEY=XXX
export ETCDBACKUP_AWS_SECRET_KEY=YYY

/usr/local/bin/etcd-backup -aws-s3-bucket rdns-etcd-backup \
            -aws-s3-region us-west-1 \
            -prefix rdnsdb \
            -etcd-v2-datadir /mnt/data/etcd
