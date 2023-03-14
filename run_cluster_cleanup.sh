#!/bin/sh
LOG_LEVEL=${LOG_LEVEL} /radix-cluster-cleanup "${COMMAND}" \
  --period=${PERIOD} \
  --cleanup-start=${CLEANUP_START} \
  --cleanup-end=${CLEANUP_END} \
  --cleanup-days=${CLEANUP_DAYS} >/dev/null