#!/bin/sh
LOG_LEVEL=${LOG_LEVEL} /radix-cluster-cleanup list-rrs-for-stop \
  --period=${PERIOD} \
  --cleanup-start=${CLEANUP_START} \
  --cleanup-end=${CLEANUP_END} \
  --cleanup-days=${CLEANUP_DAYS} > stopped_apps.txt

LOG_LEVEL=${LOG_LEVEL} /radix-cluster-cleanup list-rrs-for-deletion \
  --period=${PERIOD} \
  --cleanup-start=${CLEANUP_START} \
  --cleanup-end=${CLEANUP_END} \
  --cleanup-days=${CLEANUP_DAYS} > deleted_apps.txt