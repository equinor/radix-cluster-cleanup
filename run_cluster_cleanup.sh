#!/bin/sh
/radix-cluster-cleanup list-rrs-for-stop \
  --period=${PERIOD} \
  --cleanup-start=${CLEANUP_START} \
  --cleanup-end=${CLEANUP_END} \
  --cleanup-days=${CLEANUP_DAYS}

/radix-cluster-cleanup list-rrs-for-deletion \
  --period=${PERIOD} \
  --cleanup-start=${CLEANUP_START} \
  --cleanup-end=${CLEANUP_END} \
  --cleanup-days=${CLEANUP_DAYS}